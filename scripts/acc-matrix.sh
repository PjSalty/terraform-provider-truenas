#!/usr/bin/env bash
#
# scripts/acc-matrix.sh, run the acceptance suite against every
# TrueNAS test version we have credentials for and report a summary.
# Used to validate the multi-version compat matrix that gates v2.0.0
# final.
#
# Looks for these .envrc.local-<version> files (any subset works):
#   .envrc.local          , primary test box
#   .envrc.local-25-04    , TrueNAS SCALE 25.04 (last REST-only)
#   .envrc.local-26-beta  , TrueNAS 26.0.0-BETA.1 (REST removed)
#
# A version whose env file is absent is SKIPPED, and the summary says so
# by name. That matters: the point of this script is cross-version
# coverage, and a run that found one env file still prints per-version
# PASS lines. Without the skip list, "acc matrix summary: PASS" reads as
# "the matrix passed" when it may mean "one of three versions passed".
#
# Per-version reports go to logs/acc-matrix-<version>-<timestamp>.log.
# Exit code is non-zero if any version reports acc failures. A skipped
# version is not a failure, it is missing coverage, and the two are
# reported separately rather than collapsed.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
cd "$ROOT"

mkdir -p logs

# Every version this matrix is meant to cover. Keep the labels and the
# skip reporting in step: a version added here but never given an env
# file shows up as SKIPPED rather than quietly not existing.
KNOWN=(
  "primary:.envrc.local"
  "25-04:.envrc.local-25-04"
  "26-beta:.envrc.local-26-beta"
)

VERSIONS=()
SKIPPED=()
for known in "${KNOWN[@]}"; do
  if [ -f "${known#*:}" ]; then
    VERSIONS+=("$known")
  else
    SKIPPED+=("${known%%:*}")
  fi
done

if [ ${#VERSIONS[@]} -eq 0 ]; then
  echo "No .envrc.local-* files found, populate at least one before running."
  echo "See .envrc.local-25-04.template and .envrc.local-26-beta.template"
  exit 1
fi

echo "==> multi-version acc matrix"
echo "    running ${#VERSIONS[@]} of ${#KNOWN[@]} known version(s): ${VERSIONS[*]}"
if [ ${#SKIPPED[@]} -gt 0 ]; then
  echo "    SKIPPED (no env file, NOT covered by this run): ${SKIPPED[*]}"
fi
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
  # ACC_ENV_FILE tells acc.sh's acc_load_env which env file to source -
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
if [ ${#SKIPPED[@]} -gt 0 ]; then
  for label in "${SKIPPED[@]}"; do
    echo "  ${label}: SKIPPED (no env file, this version was NOT tested)"
  done
fi
echo ""
echo "  coverage: ${#VERSIONS[@]} of ${#KNOWN[@]} known versions actually ran"
echo ""
echo "  passed: ${PASS}"
echo "  failed: ${FAIL}"
echo ""

[ "${FAIL}" -gt 0 ] && exit 1
exit 0
