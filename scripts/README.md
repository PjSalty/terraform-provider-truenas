# scripts/

Local acceptance test runner for the TrueNAS provider. Major-provider
projects run their full test suite in CI against a managed test
environment; this provider runs it locally against an operator-owned
test TrueNAS so the test pace and the test VM stay under your control.

## One-time setup

```sh
cp .envrc.example .envrc.local
$EDITOR .envrc.local            # fill in TRUENAS_URL + TRUENAS_API_KEY
source .envrc.local             # or use direnv: `direnv allow`
```

The `.envrc.local` file is gitignored. Fetch the API key from your
secret store at shell-startup time rather than hardcoding it; the
template has the SOPS one-liner.

## The five-second sanity check

```sh
./scripts/acc-preflight.sh
```

Verifies the test TrueNAS is reachable, the API key works, and the
test pool exists. No state changes; safe to run any time. Use this
when triaging a "the suite is failing every test" report.

## The full local run

```sh
./scripts/acc.sh
# or: make acc
```

Runs the full six-stage pipeline:

| # | Stage | Cost | What it catches |
|---|---|---|---|
| 1 | Pre-flight | ~5s | Bad URL, expired API key, missing pool |
| 2 | `go build ./...` | ~5s | Syntax / unresolved imports |
| 3 | `gofmt -l` + `go vet` | ~5s | Format drift, vet warnings |
| 4 | Unit tests + 100% coverage gate | ~7m | Logic bugs in non-acc code |
| 5 | Static invariants | ~1s | Missing PreCheck / CheckDestroy / ImportState / Read-on-404 |
| 6 | Live acceptance suite (TF_ACC=1) | ~30-90m | Wire-format drift, real resource lifecycle bugs |

Each stage exits non-zero on failure with a clear message, so you see
the actual fault — not a cascade of downstream errors.

The acceptance stage streams to stdout AND saves to
`acc-YYYYMMDD-HHMMSS.log` so you can grep through a failed run after
the fact. A per-test PASS / FAIL / SKIP summary is printed at the
bottom regardless of overall outcome.

## Faster iteration loops

```sh
# Cheap (~5min): everything except the live suite
./scripts/acc.sh --skip-acc

# Just the live suite (assumes cheap stages already passed)
./scripts/acc.sh --acc-only

# Single resource
./scripts/acc.sh --resource Dataset             # → TestAccDataset*
./scripts/acc.sh --resource Certificate         # → TestAccCertificate*
./scripts/acc.sh --resource 'Dataset|Zvol'      # → both

# Just the out-of-band-delete recovery tests
./scripts/acc-disappears.sh
```

## Pre-tag checklist

Before tagging a release, run:

```sh
./scripts/acc.sh                  # full six-stage pipeline
```

A green run end-to-end means every layer that the static invariants
gate plus every behavioral test that touches the live TrueNAS has
passed against your test VM. The full log is saved to `acc-*.log` in
case you want to attach it to a release note.

## Safety rail: never against production

The acceptance suite creates and destroys real resources. Pointing
it at a production TrueNAS by accident would orphan acc-test
fixtures at best and destroy real data at worst. The runner enforces
a denylist at three layers so a typo cannot get you there:

1. **`.envrc.example`** — `TRUENAS_URL` defaults to an empty string,
   not the production hostname. You have to consciously set it to
   your test instance.
2. **`scripts/lib/_env.sh`** — `acc_assert_not_prod` runs as part of
   the env-load phase. If `TRUENAS_URL`'s hostname is in
   `TRUENAS_PROD_DENY` (default: the homelab prod TrueNAS), the
   pipeline refuses to start.
3. **`internal/acctest/acctest.Client()`** — same check, in Go. Any
   `_disappears` helper that goes through the framework hits this
   before constructing a real client, so even running `go test`
   directly without the shell runner cannot bypass the guard.

To intentionally target a host in the deny list (you should
basically never do this), export `TRUENAS_PROD_DENY=""` explicitly
and re-run. The denylist is comma-separated; add more prod hosts
in your `.envrc.local` if you have multiple.

## What the runner does NOT do

- It does not push anything anywhere. The runner is purely local.
- It does not modify your test TrueNAS outside the resources the
  acceptance suite explicitly creates and tears down. The sweeper
  (`go test -sweep=all`) cleans up any fixtures the suite leaks on
  failure; run it manually before re-running the suite if a previous
  run left state behind.
- It does not run against production. Pre-flight does a `system.info`
  call but no mutating endpoints. Pointing this at prod by mistake
  is still bad practice but cannot cause writes during the pre-flight
  stage.
