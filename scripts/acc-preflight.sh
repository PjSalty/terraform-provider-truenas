#!/usr/bin/env bash
#
# scripts/acc-preflight.sh — run only the pre-flight checks against
# the test TrueNAS. Useful when you want to confirm credentials work
# without sitting through a 2-hour acceptance run, or when triaging
# a "the acc suite is failing every test" report.
#
# Usage:
#   ./scripts/acc-preflight.sh
#
# Exits 0 on full pass, non-zero on first failure with a diagnostic
# message indicating which check failed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/_env.sh
source "${SCRIPT_DIR}/lib/_env.sh"

acc_load_env
acc_check_url
acc_check_auth
acc_check_pool

acc_ok "preflight green — test TrueNAS is reachable, authenticated, and ready"
