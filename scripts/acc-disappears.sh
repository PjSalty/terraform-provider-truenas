#!/usr/bin/env bash
#
# scripts/acc-disappears.sh — run only the *_disappears acceptance
# tests. These are the out-of-band-delete recovery tests that prove
# the resource's Read handler properly removes from state on 404.
#
# Useful when:
#   - Adding a new _disappears test and you want to iterate on just it
#   - Verifying a Read-path change did not regress drift recovery
#   - Sanity-checking before tagging a release
#
# Usage:
#   ./scripts/acc-disappears.sh
#
# Wraps scripts/acc.sh with -run '_disappears' so all the env loading,
# preflight, and per-test summary still apply.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/acc.sh" --acc-only --resource '.*_disappears'
