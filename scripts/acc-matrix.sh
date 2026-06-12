#!/usr/bin/env bash
#
# scripts/acc-matrix.sh — run the acceptance suite against every
# TrueNAS test version we have credentials for and report a summary.
# Used to validate the multi-version compat matrix that gates v2.0.0
# final.
#
# Looks for these .envrc.local-<version> files (any subset works):
#   .envrc.local           — primary target (currently 25.10.0)
#   .envrc.local-25-04     — TrueNAS SCALE 25.04 (last REST-only)
#   .envrc.local-26-beta   — TrueNAS 26.0.0-BETA.1 (REST removed)
#
# Per-version reports go to logs/acc-matrix-<version>-<timestamp>.log.
# Exit code is non-zero if any version reports acc failures.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
cd "$ROOT"

mkdir -p logs

VERSIONS=()
[ -f .envrc.local ]         && VERSIONS+=("primary:.envrc.local")
[ -f .envrc.local-25-04 ]   && VERSIONS+=("25-04:.envrc.local-25-04")
[ -f .envrc.local-26-beta ] && VERSIONS+=("26-beta:.envrc.local-26-beta")

if [ ${#VERSIONS[@]} -eq 0 ]; then
  echo "No .envrc.local-* files found — populate at least one before running."
  echo "See .envrc.local-25-04.template and .envrc.local-26-beta.template"
  exit 1
fi

echo "==> multi-version acc matrix"
echo "    found ${#VERSIONS[@]} version(s): ${VERSIONS[*]}"
echo ""

PASS=0
FAIL=0
SUMMARY=""

for entry in "${VERSIONS[@]}"; do
  label="${entry%%:*}"
  envfile="${entry#*:}"
  stamp="$(date -u +%Y%m%d-%H%M%S)"
  logfile="logs/acc-matrix-${label}-${stamp}.log"

  echo "===================================="
  echo "  running acc against ${label}"
  echo "    env: ${envfile}"
  echo "    log: ${logfile}"
  echo "===================================="

  # Run in a subshell so each version's env doesn't bleed into the next.
  # ACC_ENV_FILE tells acc.sh's acc_load_env which env file to source —
  # without it, acc.sh re-sources .envrc.local and silently re-targets
  # the primary test VM.
  if (
    export ACC_ENV_FILE="${PWD}/${envfile}"
    ./scripts/acc.sh --acc-only
  ) > "${logfile}" 2>&1; then
    echo "    ${label}: PASS"
    PASS=$((PASS + 1))
    SUMMARY="${SUMMARY}\n  ${label}: PASS"
  else
    echo "    ${label}: FAIL (see ${logfile})"
    FAIL=$((FAIL + 1))
    # surface a tail of the log for fast triage
    echo "    --- tail ${logfile} ---"
    tail -20 "${logfile}" | sed 's/^/      /'
    SUMMARY="${SUMMARY}\n  ${label}: FAIL"
  fi
  echo ""
done

echo "===================================="
echo "  acc matrix summary"
echo "===================================="
echo -e "${SUMMARY}"
echo ""
echo "  passed: ${PASS}"
echo "  failed: ${FAIL}"
echo ""

[ "${FAIL}" -gt 0 ] && exit 1
exit 0
