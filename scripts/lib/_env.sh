# shellcheck shell=bash
#
# scripts/lib/_env.sh — shared env-loading and sanity helpers used by
# every script under scripts/acc*.sh.
#
# Sourced (not executed) by the entry-point scripts. Exposes:
#   acc_load_env       — load .envrc.local + assert required vars exist
#   acc_check_url      — TCP-connect check on TRUENAS_URL
#   acc_check_auth     — `system.info` round-trip with the API key
#   acc_check_pool     — verify TRUENAS_TEST_POOL exists on the host
#   acc_color_*        — small color helpers for status output

set -euo pipefail

# ANSI colors. Honor NO_COLOR; otherwise default to color in a TTY.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  acc_color_red() { printf '\033[31m%s\033[0m' "$*"; }
  acc_color_green() { printf '\033[32m%s\033[0m' "$*"; }
  acc_color_yellow() { printf '\033[33m%s\033[0m' "$*"; }
  acc_color_bold() { printf '\033[1m%s\033[0m' "$*"; }
else
  acc_color_red() { printf '%s' "$*"; }
  acc_color_green() { printf '%s' "$*"; }
  acc_color_yellow() { printf '%s' "$*"; }
  acc_color_bold() { printf '%s' "$*"; }
fi

acc_die() {
  printf '%s %s\n' "$(acc_color_red '[acc]')" "$*" >&2
  exit 1
}

acc_info() {
  printf '%s %s\n' "$(acc_color_bold '[acc]')" "$*"
}

acc_ok() {
  printf '%s %s\n' "$(acc_color_green '[acc ok]')" "$*"
}

acc_warn() {
  printf '%s %s\n' "$(acc_color_yellow '[acc warn]')" "$*"
}

# repo_root locates the repository root by walking up from the script's
# own location. Avoids reliance on cwd so make/direnv invocations work.
acc_repo_root() {
  # Resolve scripts/lib/_env.sh -> repo root (two parents up).
  local self
  self="$(readlink -f "${BASH_SOURCE[0]}")"
  dirname "$(dirname "$(dirname "${self}")")"
}

# acc_load_env sources .envrc.local if present, then asserts the
# required vars are set. Exits non-zero with a clear message on any
# missing piece so an operator who forgot to source .envrc.local gets
# a hint instead of a confusing dial failure 30 minutes into the run.
acc_load_env() {
  local root
  root="$(acc_repo_root)"
  if [ -f "${root}/.envrc.local" ]; then
    # shellcheck disable=SC1091
    set -a
    . "${root}/.envrc.local"
    set +a
  fi
  if [ -z "${TRUENAS_URL:-}" ]; then
    acc_die "TRUENAS_URL is not set. Copy .envrc.example to .envrc.local, fill in your test host, and re-run."
  fi
  if [ -z "${TRUENAS_API_KEY:-}" ]; then
    acc_die "TRUENAS_API_KEY is not set. Inject from your secret store in .envrc.local."
  fi
  : "${TRUENAS_INSECURE_SKIP_VERIFY:=false}"
  : "${TRUENAS_TEST_POOL:=tank}"
  : "${TRUENAS_PROD_DENY:=truenas.example.com}"
  export TRUENAS_INSECURE_SKIP_VERIFY TRUENAS_TEST_POOL TRUENAS_PROD_DENY

  # Safety rail #1 — refuse to run against a known production host.
  # The acc suite creates and destroys REAL resources; pointing it
  # at prod by accident is a class of mistake this check exists to
  # prevent. The denylist is comma-separated; an exact case-
  # insensitive hostname match against the URL's authority is enough
  # to fail.
  acc_assert_not_prod
}

# acc_url_host extracts the hostname (without port) from TRUENAS_URL.
# Used by the prod-deny check; isolated as its own function so the
# parser is testable without firing the whole load path.
acc_url_host() {
  # Strip scheme, then strip path, then strip port. Tolerates the
  # absence of any of those.
  local hostport=${TRUENAS_URL#*://}
  hostport=${hostport%%/*}
  printf '%s' "${hostport%%:*}"
}

# acc_assert_not_prod fails loudly if TRUENAS_URL's hostname appears
# in TRUENAS_PROD_DENY. Comma- or whitespace-separated, case
# insensitive.
acc_assert_not_prod() {
  if [ -z "${TRUENAS_PROD_DENY:-}" ]; then
    return 0
  fi
  local host
  host="$(acc_url_host | tr '[:upper:]' '[:lower:]')"
  if [ -z "${host}" ]; then
    acc_die "could not parse hostname from TRUENAS_URL=${TRUENAS_URL}"
  fi
  # Walk the denylist. Split on comma OR whitespace.
  local deny
  for deny in $(printf '%s' "${TRUENAS_PROD_DENY}" | tr ',' ' '); do
    deny="$(printf '%s' "${deny}" | tr '[:upper:]' '[:lower:]' | xargs)"
    [ -z "${deny}" ] && continue
    if [ "${host}" = "${deny}" ]; then
      acc_die "TRUENAS_URL points at ${host}, which is in TRUENAS_PROD_DENY. \
The acceptance suite creates and destroys real resources; running it \
against this host would damage production. \
Set TRUENAS_URL to your TEST TrueNAS instance and re-run. \
To intentionally target this host (very rare), explicitly unset \
TRUENAS_PROD_DENY first."
    fi
  done
  acc_ok "TRUENAS_URL=${host} is not in TRUENAS_PROD_DENY"
}

# acc_curl is a thin wrapper around curl that respects
# TRUENAS_INSECURE_SKIP_VERIFY (silent --insecure when "true"), times
# out at 10s, and always sends the Bearer token. Stdout is the response
# body; non-zero exit means curl could not complete the request.
acc_curl() {
  local insecure=()
  if [ "${TRUENAS_INSECURE_SKIP_VERIFY:-false}" = "true" ]; then
    insecure=(--insecure)
  fi
  curl --silent --show-error --max-time 10 \
    -H "Authorization: Bearer ${TRUENAS_API_KEY}" \
    -H "Accept: application/json" \
    "${insecure[@]}" \
    "$@"
}

# acc_check_url verifies TCP reachability of TRUENAS_URL. Distinct from
# acc_check_auth because a DNS / firewall problem produces a different
# error class than a credentials problem, and the runbook reads cleaner
# when the script names the actual failure.
acc_check_url() {
  acc_info "checking TRUENAS_URL reachability: ${TRUENAS_URL}"
  if ! acc_curl --output /dev/null --head --fail "${TRUENAS_URL}/api/v2.0/system/info" 2>/dev/null; then
    # --fail rejects 4xx/5xx; do a second attempt that accepts any
    # status so we distinguish "host unreachable" from "host
    # reachable but auth wrong" (which acc_check_auth will surface).
    if ! acc_curl --output /dev/null --head "${TRUENAS_URL}/api/v2.0/system/info" 2>/dev/null; then
      acc_die "cannot reach ${TRUENAS_URL}. Check VPN, firewall, and DNS."
    fi
  fi
  acc_ok "TRUENAS_URL reachable"
}

# acc_check_auth makes a real GET /api/v2.0/system/info call and
# verifies the response is well-formed JSON with a `version` field.
# Catches expired API keys, revoked tokens, and the rare case where
# the URL is reachable but pointing at the wrong instance.
acc_check_auth() {
  acc_info "checking API key against /system/info"
  local resp
  if ! resp="$(acc_curl --fail "${TRUENAS_URL}/api/v2.0/system/info")"; then
    acc_die "/system/info request failed. API key invalid, revoked, or scoped too narrowly."
  fi
  local version
  # TrueNAS SCALE 25.10+ pretty-prints JSON with whitespace after the
  # colon (`"version": "25.10.0"`); 25.04 and earlier emitted it
  # compactly. Accept either by tolerating optional whitespace.
  version="$(printf '%s' "${resp}" | grep -oE '"version":[[:space:]]*"[^"]+"' | head -1 | sed -E 's/.*"version":[[:space:]]*"//;s/"$//')"
  if [ -z "${version}" ]; then
    acc_die "/system/info response did not include a version field. Unexpected upstream shape."
  fi
  acc_ok "authenticated as TrueNAS ${version}"
}

# acc_check_pool confirms TRUENAS_TEST_POOL actually exists. A typo
# here would let the acc suite kick off and fail every dataset/zvol
# test at create-time with a cryptic "pool not found" error.
acc_check_pool() {
  acc_info "checking TRUENAS_TEST_POOL=${TRUENAS_TEST_POOL} exists on the host"
  local resp
  if ! resp="$(acc_curl --fail "${TRUENAS_URL}/api/v2.0/pool")"; then
    acc_die "/pool listing failed. The API key may not have permission to read pools."
  fi
  # Same whitespace-tolerant match as the version grep — TrueNAS 25.10+
  # pretty-prints with a space after the colon.
  if ! printf '%s' "${resp}" | grep -qE "\"name\":[[:space:]]*\"${TRUENAS_TEST_POOL}\""; then
    acc_die "pool ${TRUENAS_TEST_POOL} not found on host. Set TRUENAS_TEST_POOL to an existing pool name."
  fi
  acc_ok "pool ${TRUENAS_TEST_POOL} exists"
}
