#!/usr/bin/env bash
#
# scripts/acc.sh — the canonical local acceptance test runner.
#
# Runs the test pipeline in the order an operator wants to see them
# fail in: cheap checks first, then live-VM checks. Each stage exits
# non-zero on failure so the operator sees the actual fault, not a
# cascade of downstream errors.
#
# Stages:
#   1. Pre-flight (TRUENAS_URL reachable, API key works, pool exists)
#   2. Build         (catch syntax / unresolved imports)
#   3. Static checks (gofmt, go vet, golangci-lint via `make lint`)
#   4. Unit tests + 100% coverage gate (`make test`)
#   5. Static invariants (all internal/provider/*invariant*_test.go)
#   6. Full acceptance suite against the test TrueNAS (TF_ACC=1)
#
# The 6-stage layout mirrors what major-provider CI workflows run
# against their test environments, but locally so the operator owns
# the test pace and the test VM.
#
# Usage:
#   ./scripts/acc.sh                  # all stages
#   ./scripts/acc.sh --skip-acc       # stages 1-5 only (cheap; ~5min)
#   ./scripts/acc.sh --acc-only       # stage 6 only (assume rest is green)
#   ./scripts/acc.sh --resource NAME  # stage 6 limited to TestAcc${NAME}*

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "${SCRIPT_DIR}")"
# shellcheck source=lib/_env.sh
source "${SCRIPT_DIR}/lib/_env.sh"

# Argument parsing ------------------------------------------------------

SKIP_ACC=0
ACC_ONLY=0
RESOURCE=""
while [ $# -gt 0 ]; do
  case "$1" in
    --skip-acc) SKIP_ACC=1; shift ;;
    --acc-only) ACC_ONLY=1; shift ;;
    --resource) RESOURCE="$2"; shift 2 ;;
    --resource=*) RESOURCE="${1#--resource=}"; shift ;;
    -h|--help)
      sed -n '3,/^$/p' "$0" | sed 's/^# \?//'
      exit 0 ;;
    *) acc_die "unknown arg: $1 (try --help)" ;;
  esac
done

cd "${REPO_ROOT}"

# Always load env so even --skip-acc surfaces a missing-config error
# early. (Some stages — like the static invariants — touch test files
# that reference env vars even when TF_ACC isn't set.)
acc_load_env

# Stage 1 — pre-flight --------------------------------------------------

if [ "${ACC_ONLY}" -eq 0 ] && [ "${SKIP_ACC}" -eq 0 ]; then
  acc_info "stage 1/6: preflight"
  acc_check_url
  acc_check_auth
  acc_check_pool
fi

# Stage 2 — build -------------------------------------------------------

if [ "${ACC_ONLY}" -eq 0 ]; then
  acc_info "stage 2/6: go build ./..."
  go build ./...
  acc_ok "build clean"
fi

# Stage 3 — static checks (lint) ----------------------------------------

if [ "${ACC_ONLY}" -eq 0 ]; then
  acc_info "stage 3/6: gofmt + go vet"
  if [ -n "$(gofmt -l . 2>/dev/null)" ]; then
    gofmt -l .
    acc_die "gofmt: files above need formatting (run: make fmt)"
  fi
  go vet ./...
  acc_ok "lint clean"
fi

# Stage 4 — unit tests + tiered coverage gate --------------------------
#
# The v2.0 WS cutover changed the coverage model: the resource and
# datasource layers previously hit 100% via REST httptest mocks, but
# wsclient-based resource code can only be exercised via live WS
# fixtures (acc tests, not unit tests). So we hold the high bar on
# the low-level packages (types, validators, wsclient, redactor) and
# accept the acc suite as the canonical coverage source for the
# resource/datasource/provider layers.

if [ "${ACC_ONLY}" -eq 0 ]; then
  acc_info "stage 4/6: unit tests + tiered coverage gate"
  if ! go test -race -timeout 15m -coverprofile=coverage.out ./...; then
    acc_die "unit tests failed"
  fi

  # Tier 1 — packages required to hold 100% (low-level, pure functions).
  # These have no live-API dependency; any regression IS a real coverage
  # loss the unit suite must catch.
  declare -A TIER1=(
    [github.com/PjSalty/terraform-provider-truenas/internal/types]=100
    [github.com/PjSalty/terraform-provider-truenas/internal/validators]=100
    [github.com/PjSalty/terraform-provider-truenas/internal/resourcevalidators]=100
    [github.com/PjSalty/terraform-provider-truenas/internal/planhelpers]=100
    [github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers]=100
    [github.com/PjSalty/terraform-provider-truenas/internal/flex]=100
    [github.com/PjSalty/terraform-provider-truenas/internal/acctest]=100
    [github.com/PjSalty/terraform-provider-truenas/internal/wsclient]=98
  )

  # Tier 2 — packages with degraded unit coverage post-WS-cutover; acc
  # is canonical. Track the floor so a future polish PR can reclaim
  # the gap by porting unit fixtures to wsclient testserver.
  declare -A TIER2_FLOOR=(
    [github.com/PjSalty/terraform-provider-truenas/internal/resources]=40
    [github.com/PjSalty/terraform-provider-truenas/internal/datasources]=35
    [github.com/PjSalty/terraform-provider-truenas/internal/provider]=45
    [github.com/PjSalty/terraform-provider-truenas/internal/recordreplay]=80
    [github.com/PjSalty/terraform-provider-truenas/internal/fwresource]=80
  )

  # Aggregate per-package coverage from coverage.out (a real run's
  # data, not -count=0 dry-run output which reports 0% across the
  # board). For each tracked package, sum the covered vs total
  # statements across its .go files.
  fail=0
  declare -A PKG_COV
  while IFS= read -r row; do
    # coverage.out file paths are ALREADY the full module-qualified
    # package path (github.com/PjSalty/.../internal/types/foo.go)
    # so dirname alone yields the canonical package key.
    pkg=$(awk -F: '{print $1}' <<<"$row" | xargs dirname)
    # coverage.out lines: <file>:<startLine>.<col>,<endLine>.<col> <numStmts> <covered>
    stmts=$(awk '{print $2}' <<<"$row")
    covered=$(awk '{print $3}' <<<"$row")
    if [ -z "${PKG_COV[$pkg]:-}" ]; then
      PKG_COV[$pkg]="0 0"
    fi
    s=$(awk '{print $1}' <<<"${PKG_COV[$pkg]}")
    c=$(awk '{print $2}' <<<"${PKG_COV[$pkg]}")
    s=$((s + stmts))
    if [ "$covered" -gt 0 ]; then c=$((c + stmts)); fi
    PKG_COV[$pkg]="$s $c"
  done < <(tail -n +2 coverage.out)

  for pkg in "${!PKG_COV[@]}"; do
    s=$(awk '{print $1}' <<<"${PKG_COV[$pkg]}")
    c=$(awk '{print $2}' <<<"${PKG_COV[$pkg]}")
    if [ "$s" -eq 0 ]; then continue; fi
    cov_int=$((c * 100 / s))
    if [ -n "${TIER1[$pkg]:-}" ]; then
      floor=${TIER1[$pkg]}
      if [ "$cov_int" -lt "$floor" ]; then
        acc_info "  TIER1 $pkg: ${cov_int}% < ${floor}% floor"
        fail=$((fail + 1))
      fi
    elif [ -n "${TIER2_FLOOR[$pkg]:-}" ]; then
      floor=${TIER2_FLOOR[$pkg]}
      if [ "$cov_int" -lt "$floor" ]; then
        acc_info "  TIER2 $pkg: ${cov_int}% < ${floor}% floor"
        fail=$((fail + 1))
      fi
    fi
  done

  if [ "$fail" -gt 0 ]; then
    acc_die "${fail} package(s) below their coverage floor"
  fi
  local_total="$(go tool cover -func=coverage.out | awk 'END {gsub("%",""); print $NF}')"
  acc_info "total coverage: ${local_total}%"
  acc_ok "unit tests + tiered coverage green"
fi

# Stage 5 — static invariants (separate so they get a clean status line)

if [ "${ACC_ONLY}" -eq 0 ]; then
  acc_info "stage 5/6: static invariants"
  if ! go test -count=1 -run 'Invariant|Test(Resources|Acceptance)' ./internal/provider/; then
    acc_die "static invariants failed"
  fi
  acc_ok "invariants clean"
fi

# Stage 6 — live acceptance suite ---------------------------------------

if [ "${SKIP_ACC}" -eq 1 ]; then
  acc_info "stage 6/6: SKIPPED per --skip-acc"
  acc_ok "all selected stages passed"
  exit 0
fi

acc_info "stage 6/6: acceptance suite (TF_ACC=1) — this is the slow one"

# Always re-check preflight before launching acc — credentials may have
# rotated since we started, and a 2-hour fail-on-credentials cycle is
# the worst-case operator experience this script is designed to avoid.
if [ "${ACC_ONLY}" -eq 1 ]; then
  acc_check_url
  acc_check_auth
  acc_check_pool
fi

ACCEPTANCE_ARGS=(
  -v -count=1 -timeout 120m -race
)

if [ -n "${RESOURCE}" ]; then
  ACCEPTANCE_ARGS+=( -run "TestAcc${RESOURCE}" )
  acc_info "limiting acceptance run to TestAcc${RESOURCE}*"
fi

LOG="${REPO_ROOT}/acc-$(date +%Y%m%d-%H%M%S).log"
acc_info "streaming output and saving full log to ${LOG}"

set +e
TF_ACC=1 go test "${ACCEPTANCE_ARGS[@]}" ./internal/resources/ ./internal/datasources/ 2>&1 | tee "${LOG}"
ACC_STATUS=${PIPESTATUS[0]}
set -e

# Per-test summary so the operator does not need to scroll the full log
acc_info "==== summary ===="
PASS_COUNT="$(grep -cE '^--- PASS: TestAcc' "${LOG}" 2>/dev/null || true)"
FAIL_COUNT="$(grep -cE '^--- FAIL: TestAcc' "${LOG}" 2>/dev/null || true)"
SKIP_COUNT="$(grep -cE '^--- SKIP: TestAcc' "${LOG}" 2>/dev/null || true)"
echo "  PASS:  ${PASS_COUNT}"
echo "  FAIL:  ${FAIL_COUNT}"
echo "  SKIP:  ${SKIP_COUNT}"
if [ "${FAIL_COUNT}" -gt 0 ]; then
  echo
  echo "  failures:"
  grep -E '^--- FAIL: TestAcc' "${LOG}" | sed 's/^/    /'
fi

if [ "${ACC_STATUS}" -ne 0 ]; then
  acc_die "acceptance suite failed (full log: ${LOG})"
fi
acc_ok "acceptance suite green (full log: ${LOG})"
