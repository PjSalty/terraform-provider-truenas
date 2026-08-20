#!/usr/bin/env bash
#
# scripts/api-drift.sh, report JSON-RPC methods this provider calls that
# TrueNAS removes in a later API version.
#
# Why this exists: nothing in CI previously checked that the middleware
# methods the provider calls still exist. service.start and service.stop
# were marked removed_in="v26.04" in 25.10.0 and went @private in 26.0,
# and the provider kept calling them. Nothing failed, because the code
# compiled, the unit tests used fakes, and no live 26.0 box was in CI.
# A removal is only observable against the upstream API definitions, so
# that is what this checks.
#
# Method: TrueNAS versions its API models under
# src/middlewared/middlewared/api/v<MAJOR>_<MINOR>[_<PATCH>]/. Each public
# method has a <Name>Args model. A model present in version N and absent
# in version N+1 is a removal. Normalising both sides (lowercase, drop
# separators) makes the model/method comparison exact without hand-maintained
# acronym rules: ISCSIGlobalSessionsArgs -> iscsiglobalsessions, and the
# method iscsi.global.sessions -> iscsiglobalsessions.
#
# This deliberately reports only REMOVALS between adjacent versions, not
# "method absent from version X". Absence alone is noise: the service.*
# namespace simply was not modelled before 25.10.0, so every service
# method would look missing on 25.04.
#
# Usage:
#   ./scripts/api-drift.sh            # check, human-readable
#   ./scripts/api-drift.sh --quiet    # only print findings
#
# Exit 0 when every removal is either unused or allowlisted.
# Exit 1 when the provider calls a method that a later API version removes
#        and the pair is not listed in scripts/api-drift-allow.txt.
# Exit 2 on a setup failure (no network, no git, upstream layout changed).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ALLOW_FILE="${REPO_ROOT}/scripts/api-drift-allow.txt"
MIDDLEWARE_REPO="${MIDDLEWARE_REPO:-https://github.com/truenas/middleware.git}"
API_SUBDIR="src/middlewared/middlewared/api"
PLUGIN_SUBDIR="src/middlewared/middlewared/plugins"

QUIET=0
[ "${1:-}" = "--quiet" ] && QUIET=1

say() { [ "$QUIET" -eq 1 ] || echo "$@"; }
die() { echo "api-drift: $*" >&2; exit 2; }

command -v git >/dev/null 2>&1 || die "git is required"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

say "==> fetching TrueNAS API definitions"
# Blobless + sparse: the api/ tree is a few hundred KB, the full repo is
# tens of MB. --depth 1 keeps it to a single commit.
git clone --quiet --depth 1 --filter=blob:none --sparse "$MIDDLEWARE_REPO" "$WORK/mw" \
  || die "could not clone $MIDDLEWARE_REPO (network?)"
git -C "$WORK/mw" sparse-checkout set "$API_SUBDIR" "$PLUGIN_SUBDIR" >/dev/null 2>&1 \
  || die "sparse-checkout failed"

API_DIR="$WORK/mw/$API_SUBDIR"
[ -d "$API_DIR" ] || die "upstream layout changed: $API_SUBDIR not found"

# Version dirs sort correctly lexically because upstream zero-pads the
# minor (v25_04_0 < v25_10_0 < v26_0_0). "base" is not a version.
VERSIONS="$(find "$API_DIR" -maxdepth 1 -mindepth 1 -type d -printf '%f\n' \
  | grep -E '^v[0-9]+_[0-9]+' | sort)"
[ -n "$VERSIONS" ] || die "no versioned API directories found"
say "    versions: $(echo "$VERSIONS" | paste -sd' ' -)"

# normalise turns both a model class name and a JSON-RPC method name into
# the same comparable token.
normalise() { tr '[:upper:]' '[:lower:]' | tr -d '._-'; }

# Every method name that appears in shipped provider code.
#
# Deliberately NOT anchored on `Call(ctx, "...")`. The method is not always
# a literal at the call site: the pre-25.10 service fallback picks its
# method in legacyServiceMethod() and passes a variable, so a call-site
# pattern silently missed it and this check reported a clean bill of health
# while the provider still referenced two removed methods. Matching any
# dotted lowercase string literal catches both spellings. Over-matching is
# harmless here because the result is intersected with "models upstream
# actually removed", which discards anything that is not a real method.
#
# Test files are excluded: they name removed methods on purpose (to assert
# they are NOT emitted), and those are assertions, not calls.
say "==> extracting methods this provider references"
find "$REPO_ROOT/internal" -name '*.go' ! -name '*_test.go' -print0 \
  | xargs -0 grep -hoE '"[a-z][a-z0-9_]*(\.[a-z0-9_]+)+"' \
  | tr -d '"' | sort -u > "$WORK/methods.txt"
METHOD_COUNT="$(wc -l < "$WORK/methods.txt" | tr -d ' ')"
[ "$METHOD_COUNT" -gt 0 ] || die "found 0 provider method references; the extraction pattern is wrong"
say "    $METHOD_COUNT candidate method names"

# Normalised lookup: normalised-token -> original method name.
: > "$WORK/methods_norm.txt"
while IFS= read -r m; do
  printf '%s\t%s\n' "$(printf '%s' "$m" | normalise)" "$m" >> "$WORK/methods_norm.txt"
done < "$WORK/methods.txt"

# Allowlist: "<method> <version-pair>" entries that are known and handled.
: > "$WORK/allow.txt"
if [ -f "$ALLOW_FILE" ]; then
  grep -vE '^\s*(#|$)' "$ALLOW_FILE" | awk '{print $1}' | sort -u > "$WORK/allow.txt" || true
fi

say "==> diffing adjacent API versions for removals"
FINDINGS=0
PREV=""
for V in $VERSIONS; do
  if [ -z "$PREV" ]; then PREV="$V"; continue; fi

  grep -rhoE '^class [A-Za-z0-9_]+Args' "$API_DIR/$PREV" 2>/dev/null \
    | sed 's/^class //; s/Args$//' | normalise | sort -u > "$WORK/prev.txt"
  grep -rhoE '^class [A-Za-z0-9_]+Args' "$API_DIR/$V" 2>/dev/null \
    | sed 's/^class //; s/Args$//' | normalise | sort -u > "$WORK/cur.txt"

  # Removed = in PREV, not in V.
  comm -23 "$WORK/prev.txt" "$WORK/cur.txt" > "$WORK/removed.txt"
  [ -s "$WORK/removed.txt" ] || { PREV="$V"; continue; }

  while IFS= read -r token; do
    hit="$(awk -F'\t' -v t="$token" '$1==t {print $2}' "$WORK/methods_norm.txt")"
    [ -n "$hit" ] || continue
    if grep -qxF "$hit" "$WORK/allow.txt" 2>/dev/null; then
      say "    allowlisted: $hit (removed in $V)"
      continue
    fi
    echo "FINDING: ${hit} is called by this provider and is REMOVED in ${V} (present in ${PREV})"
    FINDINGS=$((FINDINGS + 1))
  done < "$WORK/removed.txt"

  PREV="$V"
done

# Second check: methods upstream has SCHEDULED for removal.
#
# The model diff above cannot see these. A method carrying removed_in='vN'
# keeps its Args model in vN, so the model set never changes; what happens
# is that middleware sets private=True once the server reaches vN, and the
# method vanishes from /api/current and every vN+ endpoint at once.
# auth.login_with_api_key is exactly this shape: still modelled in v27_0_0,
# still gone on a v27 server. Catching it here is advance warning rather
# than post-mortem.
#
# removed_in sits inside the same @api_method(...) block as the Args model,
# so the model name identifies the method without parsing service
# namespaces (which are declared inconsistently upstream).
say "==> scanning for methods upstream has scheduled for removal"
PLUGIN_DIR="$WORK/mw/$PLUGIN_SUBDIR"
if [ -d "$PLUGIN_DIR" ]; then
  # shellcheck disable=SC2016  # $0 and the quotes below belong to awk, not the shell
  find "$PLUGIN_DIR" -name '*.py' -print0 | xargs -0 awk '
    /@api_method\(/ { inblk=1; buf=""; model="" }
    inblk { buf = buf " " $0
            if (model == "" && match($0, /[A-Za-z0-9_]+Args/)) model = substr($0, RSTART, RLENGTH)
          }
    inblk && /^[[:space:]]*(async[[:space:]]+)?def[[:space:]]/ {
        if (buf ~ /removed_in/ && model != "") {
          v = buf; sub(/.*removed_in[[:space:]]*=[[:space:]]*["\x27]/, "", v); sub(/["\x27].*/, "", v)
          print model "\t" v
        }
        inblk=0
    }
  ' | sort -u > "$WORK/scheduled.txt"

  while IFS="$(printf '\t')" read -r model ver; do
    [ -n "$model" ] || continue
    token="$(printf '%s' "${model%Args}" | normalise)"
    hit="$(awk -F'\t' -v t="$token" '$1==t {print $2}' "$WORK/methods_norm.txt")"
    [ -n "$hit" ] || continue
    if grep -qxF "$hit" "$WORK/allow.txt" 2>/dev/null; then
      say "    allowlisted: $hit (scheduled for removal in $ver)"
      continue
    fi
    echo "FINDING: ${hit} is called by this provider and upstream marks it removed_in=${ver}"
    FINDINGS=$((FINDINGS + 1))
  done < "$WORK/scheduled.txt"
else
  say "    (plugins tree unavailable, skipping scheduled-removal scan)"
fi

# Third check: methods that do not exist upstream AT ALL.
#
# Neither check above can see these. The removal diff only fires on a
# model that was present in one version and absent in the next, so a
# method that was never in the versioned model system, or that died
# before it existed, produces no removal event. truenas_system_update is
# exactly that: its five update.* methods went away during the 25.10.0
# config-service migration, v25_04_2 has no update.py at all, and the
# resource has therefore been dead on every version this provider
# supports without anything noticing.
#
# Middleware auto-generates the CRUD/config verbs from its CRUDService
# and ConfigService base classes without per-method Args models, so those
# are exempt or this check is 46% false positives. Everything else needs
# a model, because @api_method cannot register without one.
say "==> checking every referenced method exists in the newest API version"
NEWEST="$(echo "$VERSIONS" | tail -1)"
grep -rhoE '^class [A-Za-z0-9_]+Args' "$API_DIR/$NEWEST" 2>/dev/null \
  | sed 's/^class //; s/Args$//' | normalise | sort -u > "$WORK/newest.txt"

while IFS= read -r m; do
  # Auto-generated verbs carry no dedicated model.
  case "${m##*.}" in
    query|get_instance|create|update|delete|config) continue ;;
  esac
  # Not method names: hostnames in docs/examples, and version strings
  # like v26.04 that appear in comments.
  case "$m" in
    *.com|*.org|*.net|*.io|*.local|*.dev) continue ;;
  esac
  printf '%s' "$m" | grep -qE '(^|\.)[0-9]' && continue

  token="$(printf '%s' "$m" | normalise)"
  grep -qx "$token" "$WORK/newest.txt" && continue
  if grep -qxF "$m" "$WORK/allow.txt" 2>/dev/null; then
    say "    allowlisted: $m (no model in $NEWEST)"
    continue
  fi
  echo "FINDING: ${m} is referenced by this provider but has no API model in ${NEWEST}"
  FINDINGS=$((FINDINGS + 1))
done < "$WORK/methods.txt"

if [ "$FINDINGS" -eq 0 ]; then
  say "==> OK: every referenced method exists, and none is removed or scheduled for removal"
  exit 0
fi

echo ""
echo "$FINDINGS method(s) the provider calls are removed, or scheduled for removal, upstream."
echo "Migrate to the replacement, or add the method to ${ALLOW_FILE} with a reason"
echo "if the call is already version-guarded or the migration is tracked in an issue."
exit 1
