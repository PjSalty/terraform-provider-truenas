#!/usr/bin/env bash
#
# coverage-gate.sh, enforce the per-package coverage floors against an existing
# coverage profile.
#
# This lived in three places: the CI workflow, scripts/acc.sh stage 4, and a
# claim in the Makefile that `make test` enforced it. `make test` did not, so
# a change could show 100% locally and fail the gate in CI, which is exactly
# what happened on 2026-08-22: two branches in new code were uncovered, the
# per-package `go test -cover` reading rounded to 100.0%, and only the CI
# aggregate caught it.
#
# Usage: scripts/coverage-gate.sh [coverage-profile]
# Default profile: coverage.out
#
# Exit 0 when every tracked package meets its floor, 1 when one does not,
# 2 on a setup failure.

set -euo pipefail

PROFILE="${1:-coverage.out}"
MODULE="github.com/PjSalty/terraform-provider-truenas"

die() { echo "coverage-gate: $*" >&2; exit 2; }
[ -f "$PROFILE" ] || die "no coverage profile at $PROFILE (run the tests first)"

# Packages required to hold 100%. Low-level and pure, no live-API dependency,
# so any drop is a real coverage loss the unit suite must catch.
TIER1_PKGS="types customtypes validators resourcevalidators planhelpers
planmodifiers flex acctest wsclient sweep fwresource"

# Packages whose unit coverage was degraded by the v2.0 WebSocket cutover and
# has since been reclaimed. Same floor, tracked separately so the reason for
# the split stays visible.
TIER2_PKGS="resources datasources provider"

declare -A FLOOR
for p in $TIER1_PKGS; do FLOOR["$MODULE/internal/$p"]=100; done
for p in $TIER2_PKGS; do FLOOR["$MODULE/internal/$p"]=100; done

declare -A STMTS COVERED
while IFS= read -r row; do
  # coverage.out rows: <pkgpath>/<file>:<start>,<end> <numStmts> <count>
  pkg="$(dirname "$(cut -d: -f1 <<<"$row")")"
  n="$(awk '{print $2}' <<<"$row")"
  c="$(awk '{print $3}' <<<"$row")"
  STMTS["$pkg"]=$(( ${STMTS["$pkg"]:-0} + n ))
  [ "$c" -gt 0 ] && COVERED["$pkg"]=$(( ${COVERED["$pkg"]:-0} + n ))
done < <(tail -n +2 "$PROFILE")

fail=0
for pkg in $(printf '%s\n' "${!FLOOR[@]}" | sort); do
  s="${STMTS[$pkg]:-0}"
  if [ "$s" -eq 0 ]; then
    echo "  MISSING $pkg has no statements in the profile"
    fail=$((fail + 1))
    continue
  fi
  c="${COVERED[$pkg]:-0}"
  # Integer division on purpose: 99.95% is not 100%, and the rounded figure
  # `go test -cover` prints says otherwise.
  pct=$(( c * 100 / s ))
  if [ "$pct" -lt "${FLOOR[$pkg]}" ]; then
    echo "  FAIL $pkg: ${pct}% < ${FLOOR[$pkg]}% floor ($((s - c)) of $s statements uncovered)"
    fail=$((fail + 1))
  fi
done

total="$(go tool cover -func="$PROFILE" | awk 'END {gsub("%",""); print $NF}')"
echo "coverage-gate: total ${total}%, ${#FLOOR[@]} package(s) checked"

if [ "$fail" -gt 0 ]; then
  echo "coverage-gate: ${fail} package(s) below their floor"
  echo "Find the gap with: go tool cover -func=$PROFILE | awk '\$3!=\"100.0%\"'"
  exit 1
fi
echo "coverage-gate: OK"
