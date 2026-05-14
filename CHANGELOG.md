# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`_disappears` acceptance test coverage for every deletable resource**
  — 38 new behavioural acceptance tests in `internal/resources/*_test.go`,
  one per resource that supports out-of-band deletion. Each test creates
  the resource, deletes it via a direct API call (bypassing Terraform),
  and asserts the next plan recognises the drift with
  `ExpectNonEmptyPlan: true`. Pairs with the existing per-resource
  `CheckDestroy` callback to verify both the Terraform-driven destroy
  path and the recovery-from-deletion path. Resources covered include
  the storage family (dataset, zvol, share_nfs, share_smb,
  snapshot_task, scrub_task, replication), identity (user, group,
  api_key, privilege, keychain_credential), tasks and networking
  (cronjob, init_script, static_route, alert_service, tunable),
  certificates and misc (certificate, acme_dns_authenticator,
  kerberos_realm, kerberos_keytab, vm, vm_device,
  filesystem_acl_template, reporting_exporter, cloud_backup, vmware),
  iSCSI (target, portal, initiator, extent, targetextent, auth), and
  NVMe-oF (host, subsys, port, host_subsys, port_subsys).

- **Four new static-analysis invariant tests** in `internal/provider/`
  that scan the Go source as strings to enforce shape-level guarantees
  across every resource:
  - `TestResourcesHaveImportStateImplemented` — every
    `ResourceWithImportState` must use the passthrough helper or carry
    an explicit `// import: custom` opt-out comment.
  - `TestResourcesRemoveFromStateOnNotFound` — every resource's `Read`
    method must call `resp.State.RemoveResource(ctx)` on `IsNotFound`,
    with an allowlist for the 18 singleton-by-design resources where
    delete-is-reset-to-default semantics apply.
  - `TestAcceptanceTestsHavePreCheckOrSkip` — every `TestAcc*` function
    must either call `testAccPreCheck(t)` or contain an explicit
    `t.Skip(...)` stub.
  - `TestAcceptanceTestsHaveCheckDestroy` — every non-`PlanOnly`,
    non-stub acceptance test must wire a real `CheckDestroy` callback.

- **Production-host deny safety rail** — `internal/acctest/acctest.go`
  now refuses to build a client targeting the configured production
  hostname. Three layers of defence: shell-level check in
  `scripts/lib/_env.sh`, Go-level `assertNotProd()` in the test client
  constructor (honours `TRUENAS_PROD_DENY` env override, empty
  disables), and explicit documentation in `scripts/README.md` and
  `.envrc.example` reminding operators to point tests at a non-prod
  TrueNAS only.

- **Local acceptance-test runner** — `scripts/acc.sh` ships a six-stage
  pipeline (preflight, build, lint, unit tests + 100% coverage check,
  static invariants, full acceptance suite) with per-run log files,
  `--skip-acc`, `--acc-only`, and `--resource <name>` flags. Make
  targets `acc`, `acc-skip`, `acc-only`, `acc-preflight`,
  `acc-disappears`, and `acc-resource RESOURCE=<name>` wrap the
  script. Designed for operator-paced runs against a non-production
  TrueNAS instance; no CI dependency.

- **14 `ExpectError` negative-path acceptance tests for validators**
  — `internal/provider/acc_validator_errors_test.go` exercises every
  wired validator with hostile input, asserting plan-time rejection
  before any API call. Covers `IPOrCIDR` (invalid IP, malformed CIDR,
  5-octet "IP", text-host CIDR, IPv6 positive control), four
  `stringvalidator.OneOf` enums (`init_script.type`,
  `init_script.when`, `nvmet_port.addr_trtype`, `iscsi_target.mode`),
  three `int64validator` bounds (`certificate.key_length`,
  `nvmet_port.addr_trsvcid` low/high), and `dns_nameserver.address`
  regex rejection. Locks the `.tf`-layer contract: removing a
  validator or changing an enum without updating callers fails the
  test. Previously the entire tree had one `ExpectError` assertion.

- **Apply-idempotency check rolled out to 5 more resources** — the
  `PostApplyPostRefresh: plancheck.ExpectEmptyPlan()` invariant now
  fires on `static_route`, `group`, `cronjob`, `tunable`, and
  `iscsi_portal` in addition to the prior `dataset`, `share_smb`,
  `user`. Each carries a `PreApply` `ExpectResourceAction`
  `Create` guard on top so a Create-becoming-Update regression also
  fires. `idempotencyCheckMinimum` ratchet bumped from 3 to 8.
  Coverage went from 5.3% to 13.8% of acc test files.

- **Three new static-analysis invariants** in `internal/provider/`:
  - `TestResourcesWithSchemaVersionHaveUpgradeState` — any resource
    that ships `Version: N` (`N > 0`) in its schema must implement
    `ResourceWithUpgradeState` and ship a `*_upgradestate_test.go`.
    Catches the highest-blast-radius mistake a provider author can
    make: schema-version bumps without a state migration, which
    silently corrupt state for existing users on apply.
  - `TestImportStateVerifyIgnoreEntriesAreDocumented` — every
    `ImportStateVerifyIgnore` field across the test tree must appear
    in an explicit `allowedIgnoreFields` registry with one-line
    rationale. Defeats the "just add it to the ignore list to make
    the test pass" anti-pattern that hides real Read/Create shape
    bugs. Current registry: 46 documented entries.
  - `TestSweepersHaveAcctestPrefixGuard` — every `sweep<Name>`
    function in `sweeper_test.go` must either call an Acctest-prefix
    helper (`sweeperHasAcctestPrefix`, `sweeperDatasetIsAcctest`,
    etc.) or carry a `// sweep-no-prefix-guard: <reason>` opt-out
    comment. Defense-in-depth alongside the `TRUENAS_PROD_DENY`
    safety rail.

- **`TestSensitiveFieldsAreMarkedSensitive` invariant** — every
  schema attribute whose name strongly implies a secret value
  (`password`, `secret`, `peersecret`, `api_key`, `privatekey`,
  `dhchap_key`, `dhchap_ctrl_key`, `v3_password`, `v3_privpassphrase`,
  `passphrase`, `client_secret`, etc.) must carry `Sensitive: true`.
  Without that flag, the framework leaks the value into terraform
  plan output, terraform show, and trace logs on every apply —
  a credential-disclosure foot-gun second only to committing the
  secret to git. All 10 current sensitive-named fields pass; the
  invariant locks the contract for every future credential field.

- **Apply-idempotency rollout: 3 → 29 acceptance tests (5.3% → 49.2%)**
  — the `ConfigPlanChecks.PostApplyPostRefresh: ExpectEmptyPlan()`
  assertion is now wired into half the acc test surface, up from
  three pattern-proof resources at the start of the rigor batch.
  Each adopting resource also carries `PreApply: ExpectResourceAction
  Create` so a Create-becoming-Update regression is caught with the
  same step. `idempotencyCheckMinimum` ratchet bumped 3 → 29.
  Rolled out to: static_route, group, cronjob, tunable, iscsi_portal,
  nvmet_subsys, nvmet_port, iscsi_initiator, init_script,
  kerberos_realm, iscsi_target (extended existing PreApply guards),
  iscsi_targetextent, nvmet_host_subsys, nvmet_port_subsys, privilege,
  share_nfs, iscsi_extent, nvmet_namespace, iscsi_auth, nvmet_host,
  api_key, snapshot_task, scrub_task, zvol, certificate, rsync_task.
  Deferred: singletons with server-side defaulting, sensitive-JSON
  resources where the API masks fields on read, beta/env-gated
  resources, and complex computed-field resources (VM, replication).

- **`TestValidatorErrorCoverage` invariant + 22 ExpectError tests**
  — `acc_validator_errors_test.go` exercises every wired validator
  with hostile input, asserting plan-time rejection before any API
  call. Coverage went from 1 to 22 tests. The new ratchet test in
  `validator_error_coverage_test.go` counts the
  `TestAccValidator_*` functions and asserts `>= 22`. Removing one
  would silently drop a plan-time guarantee, so the ratchet makes
  that visible in review.

  Tests cover: `IPOrCIDR` (5), `stringvalidator.OneOf` (4),
  `int64validator.Between` boundaries (5), `stringvalidator.LengthBetween`
  boundaries (3), `stringvalidator.RegexMatches` (1), with at least
  one test per wired validator.

- **`TestAcceptanceLifecycleCoverage` invariant — 62 resources
  lifecycle-locked** — every resource family must have all four
  CRUD phases (`_basic`, `_update`, `_import`, `_disappears`) or
  appear in `lifecycleResourceExclusions` with a per-phase rationale.
  Missing any phase leaves a regression vector that escapes detection
  until a user trips over it.

  Fired one real gap on first run:
  `ACMEDNSAuthenticator` had no import test — fixed by adding an
  `ImportState` test step to `TestAccACMEDNSAuthenticator_basic`
  in the same commit.

  Exclusions are catalogued by category: data sources, singletons
  where `disappears` is a no-op reset, sensitive-payload resources
  where `import` cannot round-trip the secret, env-gated/beta
  resources, and one test-naming alias.

## [1.10.2] - 2026-04-25

### Fixed

- **Release artifact layout for Terraform Registry** — the v1.10.1 release
  was rejected by the Registry publish API with `missing files in request
  body` for the per-platform SBOM JSON files. Two issues were resolved:
  - Per-platform SPDX SBOMs were listed in `SHA256SUMS` but the Registry
    upload flow only accepts archives + the manifest. SBOMs are now
    generated under a separate goreleaser id (`sbom`) and excluded from
    `SHA256SUMS` via `checksum.ids: [default]`. SBOMs remain attached to
    the GitHub release as standalone downloadable artifacts.
  - The Terraform Registry manifest was uploaded to the GitHub release as
    `terraform-registry-manifest.json` while `SHA256SUMS` referenced it as
    `terraform-provider-truenas_<version>_manifest.json`. The release
    `extra_files` now applies a matching `name_template` so the on-release
    filename matches the checksum entry.

  This is a release-tooling fix only; provider behaviour is unchanged from
  v1.10.1.

### Added

- **FreeBSD release binaries** — goreleaser now builds `freebsd_amd64`
  and `freebsd_arm64` archives, matching the platform set published by
  `cloudflare/terraform-provider-cloudflare`. Total binary count rises
  from 5 to 7 per release.

- **Signed-release verification documentation** — `SECURITY.md` now
  describes the manual `gpg --verify` flow for the GPG-signed
  `SHA256SUMS` file shipped with every release. The signing public key
  is committed at `docs/gpg-public-key.asc` (fingerprint
  `29A6 D319 E411 670F 561E  2B9C EC8F 6B9D 7DB7 49E7`) so users can
  verify release integrity without trusting the Terraform Registry.

### Changed

- **License: MIT → MPL-2.0** — the README has long advertised MPL-2.0
  via the license badge, but the `LICENSE` file shipped MIT text. The
  file is now the canonical Mozilla Public License v2.0, matching the
  badge and aligning with the license used by HashiCorp-maintained
  Terraform providers.

- **Documentation polish** — README installation example now pins
  `version = "~> 1.10"` (was the stale `"~> 0.4"`); contributor docs
  use GitHub-flavoured terminology (pull request) consistently.

- **Test fixtures use RFC 5737 documentation IPs** — addresses in
  `internal/client/*_test.go`, `internal/resources/*_test.go`,
  `internal/provider/acc_*_test.go`, and `internal/validators/*_test.go`
  now use `192.0.2.x` / `198.51.100.x` (the RFC-reserved
  documentation ranges) instead of arbitrary RFC 1918 addresses. Test
  behaviour is unchanged.

## [1.10.1] - 2026-04-24

### Changed

- Release pipeline and metadata refresh; no functional changes versus
  v1.10.0. GitHub Actions CI runs lint, race-enabled tests with a 100%
  coverage gate, `govulncheck`, and `tfplugindocs validate`. Goreleaser
  publishes 5 platform binaries (linux/darwin/windows × amd64/arm64,
  minus windows/arm64) plus SBOMs and a GPG-signed `SHA256SUMS`.

## [1.10.0] - 2026-04-15

### Added

- **`truenas_system_update` resource** — new singleton resource for
  controlling TrueNAS SCALE update behaviour from Terraform. Manages:
  - `auto_download` (bool, default `false`) — the primary "pin" lever.
    When disabled, TrueNAS never stages an update without a conscious
    action. Backed by `/update/set_auto_download`.
  - `train` (string, optional) — the active release train (for example
    `TrueNAS-SCALE-Fangtooth`). Validated against the live
    `/update/get_trains` list at apply time. When omitted, the provider
    reads and preserves whatever the system has configured.
  - `current_version`, `available_status`, `available_version` (all
    computed) — read-only observability into the live update state,
    surfaced on every Read so the drift guard can detect out-of-band
    UI changes.

  The resource deliberately does **not** execute updates. `terraform
  apply` will never reboot production — update execution remains a
  manual action via the UI, API, or a dedicated Ansible playbook.
  `Delete` is a no-op that only removes the resource from state,
  leaving the last-applied config in effect on the system.

  Ships with 100% statement coverage on `internal/client/system_update.go`
  and `internal/resources/system_update.go`, full docs at
  `docs/resources/system_update.md`, HCL + import examples under
  `examples/resources/truenas_system_update/`, and inclusion in the
  Configure/ImportState/error-branch coverage batches. Verified
  against the TrueNAS SCALE 25.04 OpenAPI spec.

### Changed

- `internal/provider/docs_coverage_test.go` + `acceptance_coverage_test.go`
  floors raised from 62 → 63 alongside the new resource.

No breaking changes. Safe minor upgrade from v1.9.0.

## [1.9.0] - 2026-04-15

Polish layer on top of v1.8.0: prod-smoke example workspace, Registry
landing-page rewrite in the conventional provider-docs style, tone
cleanup across docs and code comments, and a goreleaser v2 deprecation
fix. No code change; no wire-path behavior change.

### Phase M — tone and style cleanup

- **`docs/index.md`** — rewritten to match the conventional provider
  index style used by hashicorp/tls, digitalocean, cloudflare, and
  integrations/github: simple frontmatter, neutral one-line purpose,
  Example Usage with a minimal HCL block, Authentication section
  with three credential-passing patterns, Safety rails section
  covering `read_only` / `destroy_protection` and the environment-
  variable emergency brake, hand-authored Schema. No stats, no
  feature lists, no marketing language.

- **`README.md`** — opening shortened from a comma-heavy promotional
  paragraph to a single neutral sentence that states WHAT the
  provider is without selling it.

- **Code comments + CHANGELOG** — promotional comparison framing
  removed across the codebase. Comments now describe each invariant
  on its own merits ("battle-hardened" for tested guarantees,
  "standard" for established patterns, "destructive resources" for
  the relevant resource class).

- **`.goreleaser.yml`** — `archives.format: zip` → `archives.formats:
  [zip]` to resolve the goreleaser v2 deprecation warning surfaced
  by tag pipeline 7628. Output is identical; future goreleaser
  releases will eventually remove the scalar form.

### Phase L — prod-smoke example workspace

- **`examples/prod-smoke/`** — a committed, version-controlled copy
  of the phased-rollout smoke test workspace that operators run
  against their production TrueNAS to verify the provider can read
  state without any ability to mutate anything. Contains:

  - `versions.tf` — provider pin matching the `~/.terraformrc`
    dev_override (source `PjSalty/truenas`, binary staged at
    `/tmp/terraform-provider-truenas`).
  - `variables.tf` — `truenas_url`, `truenas_api_key` (sensitive),
    `smoke_dataset_pool`, `smoke_dataset_name`. Validation blocks
    on the URL (HTTPS required) and the API key (length sanity).
  - `provider.tf` — **Phase 1 rail armed**: `read_only = true`
    AND `destroy_protection = true` both set. Phase 1 is a refresh-
    only drift check: the provider can see prod but physically
    cannot mutate it. Comments walk the operator through Phase 2
    (`read_only=false`, destroy rail still armed) and Phase 3
    (brief destroy window with re-arm).
  - `main.tf` — imports ONE existing dataset into state with an
    `import { to = ... id = ... }` block and a matching
    `resource "truenas_dataset" "smoke"` stanza that the provider
    populates from the server during import-read. Zero changes
    expected on `terraform plan`; any drift surfaces exactly what
    the provider's Read path doesn't round-trip cleanly.
  - `RUN.md` — step-by-step runbook including the SOPS decrypt
    command, the env var export sequence, the expected output,
    the Phase 2 / Phase 3 transitions, and the emergency brake
    (`TRUENAS_READ_ONLY=1 TRUENAS_DESTROY_PROTECTION=1` env vars
    that override HCL).

  `terraform validate` against this workspace passes cleanly with
  the v1.8.0 binary staged at `/tmp/terraform-provider-truenas`.
  The workspace is NOT imported into any CI job — it's a manual
  operator tool.

### Phase K — 100% unit-test coverage (CI gate satisfied)

- **Every package at 100.0% statement coverage.** The CI pipeline's
  per-package 100% coverage gate now passes
  against main. Main pipelines from the v1.6.0 tag onward had been
  failing because Phase B–F additions introduced ~25 uncovered
  functions across 6 packages; this release closes every gap.

- **Functions covered**:

  - `internal/client/client.go` — `newRequestID` refactored into a
    testable `newRequestIDFrom(io.Reader)` plus a thin wrapper;
    `APIError.Error`, `Delete`, `DeleteWithBody`,
    `DefaultRetryPolicy` get targeted unit tests.
  - `internal/client/redact.go` — `redactJSONBody` dead-branch
    (re-marshal failure on Go values that came from `json.Unmarshal`)
    removed — walkRedact only emits marshalable types; `redactMessage`
    gains empty-string + fragment-at-start test coverage.
  - `internal/client/job_helper.go` — `waitIfJobResponse` gains the
    non-int sync-response test (object / string / array bodies).
  - `internal/client/client.go doOnce` — transport-error branch
    now exercised via 127.0.0.1:1 refused-connection test.
  - `internal/planhelpers/destroy_warning.go` — `WarnOnDestroy`
    gains the empty-ID fallback branch test.
  - `internal/planmodifiers/pem_equivalent.go` — `PlanModifyString`
    gains the "PEM plan + non-PEM state" branch test (the inverse
    of the pre-existing "non-PEM plan + PEM state" case).
  - `internal/resourcevalidators/required_when_equal.go` —
    `ValidateResource` gains three branch tests:
    unknown-discriminator, GetAttribute-error on discriminator,
    GetAttribute-error on a required attribute (with `continue`
    loop semantics).
  - `internal/resources/*.go` — 15 `ModifyPlan` hooks + 3
    `ConfigValidators` methods covered via a single table-driven
    test file (`phaseF_modifyplan_coverage_test.go`) that uses the
    pre-existing `callModifyPlanDelete` / `schemaOf` helpers. Each
    resource's null-plan + non-null-state call exercises its
    `WarnOnDestroy` body path; each ConfigValidators call
    dereferences the returned list and touches Description /
    MarkdownDescription.

- **No production code behavior change.** The only production
  delta is the `newRequestID` split into `newRequestID` +
  `newRequestIDFrom(io.Reader)` and the deletion of one dead
  branch in `redactJSONBody`. Both are internal to the client
  package and invisible at the wire level.

### Phase J — acceptance test coverage ratchet

- **`internal/provider/acceptance_coverage_test.go`** —
  `TestAcceptanceTestCoverage` (floor = 62). Walks
  `internal/resources/*.go`, identifies every resource file, and
  verifies its sibling `*_test.go` exists AND contains at least
  one `func TestAcc*` declaration. Fails on missing files, empty
  test files, or count below the floor.

- **`internal/resources/cloudsync_credential_test.go`** —
  the final missing acceptance test, closing 61→62 coverage.
  Shallow `PlanOnly + ExpectNonEmptyPlan` test mirroring the
  existing `TestAccCloudSync_schemaValidation` pattern: exercises
  schema compilation, HCL parsing, validators, and plan
  modifiers end-to-end without requiring live TrueNAS or external
  cloud credentials.

- **`make prod-ready`** gate extended to 23 invariants
  (Phase B+C+D+E+F+G+H+I+J).

### Phase I — docs & examples coverage ratchet

- **`internal/provider/docs_coverage_test.go`** — new static-analysis
  test file with two ratchets:

  - **`TestDocsCoverage`** — three-way cross-check between:
    1. Every resource type declared via `ProviderTypeName + "_..."`
       in `internal/resources/*.go`
    2. Every `docs/resources/*.md` registry doc
    3. Every `examples/resources/truenas_*/{resource.tf,import.sh}`
       example directory
    Fails if any resource lacks a doc or example, if any doc/example
    is orphaned (resource removed/renamed), or if the total falls
    below the `docsCoverageFloor = 62` SLO. No network, no
    tfplugindocs, no terraform — pure file-layout check.

  - **`TestDocsNoPlaceholders`** — greps every committed doc and
    example for TODO/FIXME/XXX/PLACEHOLDER/your-value-here markers.
    Fails if any scaffolding leaks into a tagged release.

- **Legacy example dirs removed** — `examples/resources/dataset/`,
  `examples/resources/iscsi/`, `examples/resources/share_nfs/` were
  stale non-prefixed duplicates from the pre-registry naming era.
  Replaced by the current `examples/resources/truenas_<type>/`
  canonical layout that tfplugindocs expects.

- **`templates/guides/`** — added to protect the 7 hand-authored
  prose guides (architecture, backup-strategy, getting-started,
  importing-existing, kubernetes-storage, phased-rollout,
  upgrade-to-v1) from destructive regeneration. `tfplugindocs
  generate` deletes guides with no corresponding template source;
  copying the guides into `templates/guides/` makes them the source
  of truth for regeneration runs.

- **`make docs`** — semantics changed from `generate` (destructive)
  to `validate` (read-only). The hand-authored docs carry custom
  `subcategory:` frontmatter and prose descriptions that
  `tfplugindocs generate` strips; defaulting to validate prevents
  accidental loss during a routine doc lint.

- **`make docs-regen`** — new target, explicitly dangerous, for
  bulk-bootstrap or schema-wide rename scenarios where a full
  regeneration is intentional. Must be followed by a careful
  diff review.

- **`make prod-ready`** gate extended to 22 invariants
  (Phase B+C+D+E+F+G+H+I).

### Phase H — strict static analysis (golangci-lint, 18 linters)

- **`.golangci.yml`** extended from 10 to 18 enabled linters. Added
  correctness and security linters: `bodyclose`, `contextcheck`,
  `copyloopvar`, `errorlint`, `gosec`, `nilerr`, `unconvert`,
  `usestdlibvars`. The existing 10 (`errcheck`, `gocritic`, `godot`,
  `govet`, `ineffassign`, `misspell`, `prealloc`, `staticcheck`,
  `unparam`, `unused`) remain. `gosec` and `usestdlibvars` are
  scoped out of `_test.go` where they dominate with false positives
  (glob-sourced `os.ReadFile`, test-fixture permissions, magic
  HTTP status codes in assertions).

- **Correctness fixes driven by the new linters**:

  - **`bodyclose` in client.go**: refactored `doOnce` to no longer
    return `*http.Response`. `parseRetryAfter` now runs inside
    `doOnce` (while the response is alive and about to be closed)
    and the parsed duration is stamped onto `APIError.retryAfter`
    before return. `doRequest`'s retry loop is simplified: it
    classifies via `errors.As(err, &apiErr)` instead of
    `resp == nil`. Callers receive bytes, never a still-open
    response — bodyclose safety is guaranteed at the caller
    boundary regardless of retry logic.

  - **`nilerr` × 5**: the recurring "TrueNAS API returns either a
    job ID or a sync-completed sentinel" pattern is now centralized
    in a single `client.waitIfJobResponse(ctx, resp, opLabel)`
    helper with a documented dual-response contract and a
    `//nolint:nilerr` annotation in exactly one place. Four
    client-side callers in `app.go`, `certificate.go`, `pool.go`
    now use the helper. The fifth case in
    `resources/cloud_backup.go` (filterJSONByKeys reference-decode
    fallback) is a different intentional pattern and gets its own
    `//nolint:nilerr` with a doc comment.

  - **`errorlint` in redact_wiring_test.go**: removed the custom
    `errorsAs` helper shim (written to avoid importing `errors`)
    and replaced with stdlib `errors.As`, which is the idiomatic
    and type-safe path for unwrapping. Test now imports `errors`.

  - **`contextcheck` in planhelpers/destroy_warning.go**: the
    `WarnOnDestroy` helper was using `context.Background()` inside
    its body instead of threading the caller's ctx through to
    `req.State.GetAttribute`. The function signature now binds
    `ctx context.Context` (was `_ context.Context`) and threads it.

  - **`copyloopvar` × 90**: deleted 90 `tc := tc` shadowing lines
    across 32 test files. Redundant since Go 1.22 (module requires
    1.25.0). A small Python helper (`/tmp/fix-copyloopvar.py`,
    one-off, not committed) refused to touch any line that didn't
    regex-match a `<name> := <name>` self-shadow.

  - **`gocritic` paramTypeCombine**: `RequiredWhenEqual` signature
    tightened from
    `func(discriminator string, trigger string, required []string)`
    to `func(discriminator, trigger string, required []string)`.

  - **`staticcheck` QF1011**: removed a redundant explicit type
    annotation on the compile-time interface assertion for
    `RequiredWhenEqual` in its test file; the constructor already
    declares the return type.

  - **`goimports` × 18**: auto-formatted imports across 18 resource
    files via `golangci-lint fmt`.

- **`make prod-ready`** gate extended to 21 invariants (Phase
  B+C+D+E+F+G+H). The new Phase H gate auto-detects
  `golangci-lint` in `$PATH` or falls back to
  `$(go env GOPATH)/bin/golangci-lint` so a fresh checkout that
  installs the linter via `go install` works out of the box.
  Full gate still <30s wall-clock including the lint run
  (previously <3s without lint; golangci-lint dominates).

### Phase G — secret redaction in error diagnostics

- **`internal/client/redact.go`** — every non-2xx response body is
  now passed through `redactJSONBody` before it lands on
  `APIError.Body`. Sensitive field values are recursively replaced
  with `[REDACTED]` based on a case-insensitive substring match of
  the JSON key against a fragment list covering `password`,
  `privatekey`, `dhchap_key`, `api_key`, `token`, `secret`,
  `auth`, `credential`, `passphrase`, common cloud-API token field
  names, and more. Non-JSON error bodies are truncated at 512 bytes with a
  `[non-JSON error body, truncated]` prefix.

- **`redactMessage`** — the parsed `message` field is scanned for
  any sensitive-key fragment substring; if found, the message is
  truncated before that fragment and a `[REDACTED]` marker appended.
  TrueNAS middlewared occasionally echoes back offending request
  fields in its Pydantic validation output; this catches that.

- **Why this matters**: `APIError.Error()` flows directly into
  `resp.Diagnostics.AddError()` on every single resource CRUD
  path (37 call sites across 10+ resource files). That diagnostic
  ends up in Terraform's plain-text stderr AND in state-file error
  annotations. Without redaction, a 422 carrying a `dhchap_key`
  or `password` echo would leak material into operator shells and
  shared state backends. The fix is applied once at the source —
  zero resource-side code changes required.

- **Invariant tests (9 total)**:
  - `TestIsSensitiveKey` — 21-case substring matcher unit test
  - `TestRedactJSONBody_{FlatObject,NestedObject,Array,NonJSON,NonJSONTruncated,Empty}`
  - `TestRedactMessage` — passthrough + truncation cases
  - `TestAPIErrorBodyNeverLeaksSecrets` — end-to-end APIError round-trip
  - `TestDoOnceRedactsAPIErrorBody` — httptest wiring test that stands up a
    real server returning a sensitive JSON body and asserts both
    `err.Error()` and `APIError.Body` are scrubbed
  - `TestDoOnceRedactsMessageField` — httptest wiring test for the
    parsed-message branch

- **`make prod-ready`** gate extended to 20 invariants (Phase B+C+D+E+F+G).

### Phase F — plan-time destroy warnings

- **`internal/planhelpers.WarnOnDestroy`** — reusable
  resource.ModifyPlan helper that emits a Warning diagnostic at
  plan time whenever a resource is about to be destroyed. The
  warning names the resource type and ID, explains the impact,
  and points at the `destroy_protection` flag for the blocking
  rail. Non-blocking (the safety rail is
  `client.DestroyProtection` — this is the "see before the cliff"
  rail that complements the "brake at the cliff" rail). Matches
  a standard pattern for destructive resources.

- **22 resources** now call `WarnOnDestroy` from their ModifyPlan
  hook: certificate, cloud_backup, cloud_sync, cronjob, dataset,
  group, init_script, iscsi_auth, iscsi_extent, iscsi_portal,
  iscsi_target, nvmet_host, pool, replication, rsync_task,
  scrub_task, share_nfs, share_smb, snapshot_task, user, vm, zvol.
  14 of those had no existing ModifyPlan and got it newly added;
  8 had an existing ModifyPlan (for other validation logic) and
  got `WarnOnDestroy` prepended ahead of their early-return on
  null plan.

- **`TestDestroyWarningCoverage`** — a SLO-style ratchet that
  fails if the count of resources carrying WarnOnDestroy drops
  below 22. Same mechanism as `TestIdempotencyCheckCoverage` and
  `TestConfigValidatorsCoverage`.

- **4 unit tests** for the helper itself:
  `TestWarnOnDestroy_DestroyEmitsWarning`,
  `TestWarnOnDestroy_CreateIsNoOp`,
  `TestWarnOnDestroy_UpdateIsNoOp`,
  `TestWarnOnDestroy_BothNullNoOp`.

- **`make prod-ready`** gate extended to 19 invariants (Phase B+C+D+E+F).
  Still <3s wall-clock, no live infra.

### Phase E — config-time cross-attribute validators

- **`internal/resourcevalidators` package** with the
  `RequiredWhenEqual` helper: when a discriminator attribute
  matches a trigger value, every required companion attribute
  must be set. Runs at config-validation time, before any network
  round-trip. 7 unit tests covering happy path, missing-both,
  missing-one, non-trigger, null discriminator, empty-string,
  and descriptions.

- **ConfigValidators wired onto three resources** with enum
  discriminators:
  - `truenas_certificate` — `create_type=CERTIFICATE_CREATE_IMPORTED`
    requires `certificate` + `privatekey`.
  - `truenas_iscsi_extent` — `type=DISK` requires `disk`,
    `type=FILE` requires `path`.
  - `truenas_network_interface` — `type=LINK_AGGREGATION` requires
    `lag_protocol`, `type=VLAN` requires `vlan_parent_interface`.

- **`TestConfigValidatorsCoverage`** ratchet (floor: 3, bump on
  every new validator).

### Phase D — destroy-protection safety rail ("safe apply" profile)

- **`client.DestroyProtection` + `ErrDestroyProtected`**: a second
  client-layer safety rail that blocks ONLY `DELETE` requests while
  allowing `GET`/`POST`/`PUT` through. Layers beneath `ReadOnly`:
  when both flags are set, `ReadOnly` dominates (strictly broader).
  When only `DestroyProtection` is set, the provider is in "safe
  apply" mode — creates and updates flow, destroys are refused at
  the wire. Matches the per-resource `deletion_protection` pattern
  found in major Terraform providers, except enforced for
  every resource in the provider at once — zero per-resource
  coverage gap.

- **Provider schema `destroy_protection` + env `TRUENAS_DESTROY_PROTECTION`**
  with HCL-precedence-over-env wiring. Defaults to false for
  backwards compatibility. Verbose tflog.Warn on every refused
  DELETE with method/path/req_id for operator correlation.

- **13 new tests**: 6 at client layer (blocks DELETE, allows
  GET/POST/PUT, disabled path, layered with ReadOnly, nil-receiver
  guard, errors.Is wrapping), 4 at provider Configure layer (env
  var table-driven, HCL attribute, HCL-overrides-env, safe-apply
  profile combo). All green in ~5ms total.

- **`make prod-ready`** gate extended to 15 invariants including
  the Phase D tests. Still <3s wall-clock, no live infra.

- **Documentation**:
  - `docs/guides/phased-rollout.md` Phase 3 is now "Safe-apply
    profile: drop read-only, keep destroy-protection" with full
    drill. Phase 3.5 covers intentional destroys with re-arming
    discipline. Emergency brake re-arms BOTH rails.
  - `README.md` has a new "Destroy-protection mode (apply-safe
    rail)" subsection with the production recipe and the
    re-arm pattern.
  - `examples/provider/provider.tf` has a second commented block
    showing the safe-apply profile alongside the read-only profile.

### Added — Phase B battle-hardening for prod rollout

- **Read-only safety rail** (`client.Client.ReadOnly` field + `ErrReadOnly`).
  When enabled, every mutating request (POST/PUT/DELETE) fails before any
  network call is made — the target TrueNAS never sees the attempt, not
  even in access logs. Configurable via `read_only = true` in the provider
  block OR the `TRUENAS_READONLY={1,true}` environment variable. HCL takes
  precedence. Intended use: `terraform plan` against production with the
  rail engaged, flip off only after the plan looks correct.
- **Fault injection tests** at the client layer: malformed JSON,
  wrong-shape responses, empty bodies on typed methods, slow bodies
  (context-deadline honored), connection reset mid-body (retry recovers),
  and raw-socket garbage (transport error surfaces, no panic). 6 tests
  in `internal/client/fault_responses_test.go`.
- **plancheck.ExpectEmptyPlan** on dataset/user/share_smb `_basic`
  acceptance tests. Catches the "terraform plan is never clean" family
  of provider bugs where Read returns values the state doesn't hold.
- **Sweeper coverage invariant** (`TestSweeperCoverage`) — every
  resource MUST either be registered with a sweeper or be in the
  `resourceSweeperExclusions` map with a rationale. Closes the silent
  38/62 gap; 24 legitimately excluded (singletons, dangerous, pending)
  with per-entry justification.
- **Apply-idempotency coverage ratchet** (`TestIdempotencyCheckCoverage`)
  — a SLO-style gate that fails if the number of acc tests with
  `PostApplyPostRefresh: ExpectEmptyPlan` drops below the floor.
  Current floor: 3; bump per-rollout.
- **Delete-NotFound invariant** (`TestDeleteHandlesNotFound`) — every
  non-singleton resource's Delete MUST call `client.IsNotFound` so a
  delete-while-already-gone race surfaces as a graceful state removal,
  not a fatal Terraform error. 15 singleton exclusions documented.
- **CRUD logging invariant** (`TestCRUDLogging`) — every resource's
  Create/Read/Update/Delete MUST emit at least one tflog call inside
  its body. Drive-by refactors can no longer silence the operator.
  Currently 248/248 CRUD methods pass.
- **Typed-CRUD readonly test** — exercises the safety rail through
  `CreateDataset` / `UpdateDataset` / `DeleteDataset` / `GetDataset`
  to prove no typed wrapper swallows `ErrReadOnly` on the way up.

### Live validation

- **Full `TF_ACC=1` acceptance run against TrueNAS SCALE 25.10.0**
  (test VM test VM): **149 PASS / 0 FAIL / 6 SKIP** across
  the 62-resource surface, wall-clock 866s. The 6 skips are
  deliberate: `KMIPConfig_update` (needs external KMIP server),
  `NetworkInterface_basic`/`_update` (writes can disconnect the
  cluster), `PoolResource_disappears` / `SystemDataset_disappears`
  (dangerous), and `CertificateResource_update` (known limitation,
  see below). Two fixture bugs were found and fixed during the run
  (`NVMetHost_update` missing `dhchap_hash`; `Certificate_update`
  PEM-normalization drift).

### Phase C — plan-modifier hygiene gaps closed

- **PEM semantic-equality plan modifier** (`internal/planmodifiers.PEMEquivalent`).
  Decodes every PEM block in plan and state values, re-encodes them
  through `encoding/pem`, and treats the two values as equal when
  their canonical forms match — even when the server has re-wrapped
  base64 lines, swapped CRLF for LF, or stripped trailing whitespace.
  Wired into `truenas_certificate.certificate` and `privatekey` so
  an in-place rename no longer tries to destroy+create on cosmetic
  normalization. Un-skips `TestAccCertificateResource_update`.

- **111 Optional+Computed attributes now carry `UseStateForUnknown()`**.
  Before this release, omitting any such attribute from HCL on a
  subsequent apply caused the Plugin Framework to mark the plan
  value as Unknown ("known after apply"), which showed up as a
  phantom diff on every plan — and for the 6 attributes that ALSO
  had `RequiresReplace()`, it falsely forced destroy+create cycles.
  One of those six (`truenas_certificate.key_type`) was the actual
  root cause of the v1.0 `TestAccCertificateResource_update`
  failure. Mass-fixed across 34 resource files: acme_dns,
  app, certificate, cloud_sync, cronjob, dataset, directoryservices,
  dns_nameserver, group, init_script, iscsi_{extent,initiator,portal,target},
  mail_config, network_{config,interface}, nfs_config,
  nvmet_{global,host,namespace,port,subsys}, pool, replication,
  rsync_task, share_{nfs,smb}, snmp_config, ups_config, user,
  vm, vm_device, zvol.

- **Two new static invariants** in `internal/provider/` block
  regressions on the above:
  - `TestRequiresReplaceRespectsUseStateForUnknown` — every
    Optional+Computed+RequiresReplace attribute MUST carry
    `UseStateForUnknown()` BEFORE `RequiresReplace()` in its
    plan-modifier slice, or the test fails with a file:attribute
    punch list.
  - `TestOptionalComputedHasUseStateForUnknown` — every
    Optional+Computed attribute without a `Default:` MUST carry
    `UseStateForUnknown()` or be in the small exclusion map
    (with a rationale). Catches the broader phantom-diff family.

- **HTTP request-ID correlation at the client layer**
  (`client.newRequestID` + `X-Request-ID` header). Every logical
  API call is tagged with a 16-char lowercase hex ID generated
  from `crypto/rand`; that ID is set on the outgoing header and
  threaded through every `tflog` breadcrumb for the call. Retries
  of the same logical operation share one ID so operators can
  correlate client-side traces with TrueNAS middlewared audit
  entries without the retry storm fragmenting the investigation.
  Covered by `TestNewRequestID_ShapeAndUniqueness`,
  `TestDoRequest_EmitsXRequestIDHeader`, and
  `TestDoRequest_RetriesShareRequestID`.

- **Phase B+C gate**: `make prod-ready` now runs the two plan-modifier
  invariants, the three request-ID tests, and the 11-test PEM
  semantic-equality suite in addition to the existing Phase B
  battle-hardening checks. Still <3s wall-clock, no live infra.

### Known limitations

- None at this release. The v1.1.0-rc `truenas_certificate`
  in-place rename gap was closed by the PEM semantic-equality
  plan modifier above, and `TestAccCertificateResource_update`
  passes in the live TF_ACC run.

## [1.0.0] - 2026-04-13

### Added — comprehensive coverage release

- **12 packages × 100.0% literal statement coverage**, race-clean:
  `main`, `cmd/skaff`, `internal/acctest`, `internal/client`,
  `internal/datasources`, `internal/flex`, `internal/fwresource`,
  `internal/planmodifiers`, `internal/provider`, `internal/resources`,
  `internal/sweep`, `internal/validators`. CI enforces this as a gate.
- **Hard fuzz regression**: 8 fuzz targets × 30s each = 52,486,918
  executions, zero crashes. Corpus persistence under `testdata/fuzz/`.
- **8 benchmarks** covering hot paths (doRequest, backoffDelay,
  4× mapResponseToModel, 2× validators).
- **Integration tests** via `resource.UnitTest` with a `mockTrueNAS`
  httptest backend — run under plain `go test`, no TF_ACC required.
- **PlanCheck assertions** in 5 representative acceptance tests
  (Create/Update actions, known values, Update-not-Replace guards).
- **tflog.Trace instrumentation**: 985 entry/exit calls across all
  resource CRUD handlers and client methods.
- **8 new data sources** (now 33 total): iscsi_target, iscsi_portal,
  iscsi_extent, iscsi_initiator, api_key, keychain_credential,
  snapshot_task, alert_service.
- **7 resources with ResourceWithModifyPlan** for cross-attribute
  validation at plan time (nvmet_host, vm, iscsi_extent, share_nfs,
  certificate, replication, iscsi_target).
- **truenas_cronjob seeds the SchemaVersion: 1 + StateUpgrader**
  pattern for future schema migrations.
- **3 new plan modifiers**: RequiresReplaceIfChangedInt64,
  RequiresReplaceIfChangedBool, JSONEquivalent.
- **4 new helper packages**: internal/flex,
  internal/acctest, internal/fwresource, internal/sweep.
- **cmd/skaff**: resource scaffolding tool with 16 unit
  tests and 100% coverage.
- **Release pipeline**: goreleaser v2 expanded to 14 platform targets
  (up from 6), SBOM via syft, GPG signing stanza, registry manifest.
- **Binary-level E2E verified**: released artifact installed via
  dev_overrides, terraform validate + plan succeed for 10 resource
  types.
- **Community infrastructure**: SECURITY.md, CODE_OF_CONDUCT.md,
  CODEOWNERS, issue/PR templates, pre-commit hooks, renovate,
  .changelog/ + changie, markdownlint, yamllint.
- **Expanded CONTRIBUTING.md** (~200 lines) with industry-standard
  workflow, resource addition checklist (incl. skaff), quality gates
  table.
- **New docs/guides**: architecture.md, upgrade-to-v1.md.
- **Coverage gate in CI** fails any drop below 100.0%.
- **test:fuzz CI job**: manual 30s-per-target smoke run.

### Fixed

- `google.golang.org/grpc` bumped v1.79.2 → v1.79.3 (fixes
  GO-2026-4762, authorization bypass via missing leading slash in
  `:path`).

### Security

- Sensitivity audit: `Sensitive: true` added to
  `reporting_exporter.attributes_json` and `vm_device.attributes`;
  18 existing credential fields verified.

---

## Prior entries — rolled into 1.0.0

- **MILESTONE: Full acceptance test suite 100% green against
  TrueNAS SCALE 25.10.0.** Final run: 151 passing + 5 intentional skips
  = 156/156, 0 failures. Skipped tests are legitimate exceptions: KMIP
  update (requires real KMIP server), network_interface basic+update
  (env-gated for safety on shared VM), pool disappears (can't destroy
  the test pool), systemdataset disappears (singleton). 2 resources
  graduated from Beta to GA based on live verification:
  `truenas_cloudsync_credential` and `truenas_kerberos_keytab`. Remaining
  7 Alpha/Beta resources are gated on infrastructure the acceptance test
  environment lacks (real DNS provider for ACME, real KDC for directory
  services, external cloud credentials, Fibre Channel for vmware).
- **Live TF_ACC run against TrueNAS SCALE 25.10.0 test VM** (test VM —
  separate from production). First cold run: 133/152 pass (87.5%). Surfaced
  and fixed several real provider bugs:
  - `truenas_dataset.comments` and `truenas_zvol.comments` now use
    `dataset.GetComments()` which transparently handles SCALE 25.10's move
    of `comments` from top-level to `user_properties.comments` (top-level
    is always null in 25.10, breaking round-trip on every dataset).
  - `truenas_certificate.disappears`: cert delete on an already-gone cert
    returns `[ENOENT] Certificate N does not exist` from the long-running
    job rather than an HTTP 404. The client `DeleteCertificate` helper now
    normalizes that to a `404` `*APIError` so resource Delete handlers can
    use `client.IsNotFound` to treat it as success.
  - `truenas_catalog.sync_on_create` now has a `booldefault.StaticBool(false)`
    so the field is always Known after Update (Terraform was rejecting plans
    with "Provider returned invalid result object after apply").
  - `truenas_vm.bootloader_ovmf` and `enable_secure_boot` are no longer
    sent in `vm.update` requests — SCALE 25.10 rejects them with HTTP 422
    "Extra inputs are not permitted". They remain Computed-readable but are
    create-time-only.
- **Unit tests expanded to 1799 passing** (up from 1079): datasource package
  gains a full httptest-based harness (`testutil_test.go`) that builds a real
  `*client.Client`, configures it with a mocked server, and invokes
  `datasource.Read` through a `tfsdk.Config` wire — exercising the full
  config-decode → client-call → state-set path. Every data source gets a schema
  test plus success/404/500/invalid-JSON/empty-list/lookup-by-name coverage.
  Resource package gains batch `Schema`/`Metadata`/`Configure`/`ImportState`
  tests plus `mapResponseToModel` fixture cases for 50+ resources and CRUD
  roundtrip tests for 8 singleton configs + 12 ID-based resources against an
  `httptest.Server`-backed client. **Package coverage: datasources 1.7% →
  73.3%, resources 7.9% → 35.8%, overall 22.5% → 48.8%.**
- **HCL validation test suite** in `internal/provider/examples_validation_test.go`
  parses every `examples/resources/truenas_*/resource.tf` via
  `github.com/hashicorp/hcl/v2/hclparse` to catch broken example syntax before
  release. Also verifies every registered resource has both a doc page
  (`docs/resources/<name>.md`) and an example directory, guarding against
  resource additions that forget to update docs.
- **goreleaser snapshot build verified offline** — `goreleaser release
  --snapshot --clean --skip=publish --skip=sign` produces signed archives for
  linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64,
  windows/arm64 in 32s. All 6 binaries ~7MB trimpath/stripped. SHA256SUMS file
  generated.
- **Sweeper functions for 38 resources** via `internal/provider/sweeper_test.go`
  using `resource.AddTestSweepers`. Run with
  `go test -v ./internal/provider/ -sweep=all` to clean up abandoned acceptance
  test fixtures. Sweepers use an `acct`/`acctest`/`tf-acc` prefix filter so they
  only touch test-managed resources, and encode parent/child ordering via
  `Sweeper.Dependencies` so leaf resources (join tables, namespaces, shares)
  are swept before their parents (datasets, vms, subsystems).
- **Hand-written docs + examples for all 62 resources.** `docs/resources/*.md`
  follows the standard Terraform Registry format (frontmatter + intro +
  Example Usage + Argument Reference + Attribute Reference + Timeouts +
  Import). Each attribute
  is documented with Required/Optional status, valid values (from schema
  validators), defaults, and `RequiresReplace` notes. `tfplugindocs validate`
  passes. Every resource has a runnable `examples/resources/truenas_<name>/`
  directory with `resource.tf` + `import.sh`.
- **~386 new attribute validators across 36 resources**: regex format
  validators (POSIX usernames, dataset names, iSCSI IQNs, SMB share names,
  octal umasks, hex OUIs, `/mnt/` dataset paths); `OneOf` enum validators for
  VM cpu_mode, certificate key_type/key_length, service names, alert providers,
  iSCSI rpm/blocksize, dataset recordsize, NVMe-oF dhchap_hash, SSH log
  levels/facilities, SMB unixcharset; int range bounds on UIDs/GIDs, VM
  cores/threads/memory, certificate lifetime, snapshot lifetime, scrub
  threshold, FTP clients/timeouts, NVMe-oF queues/inline_data; length bounds on
  ~80 string attributes; `validators.IPOrCIDR()` on network_interface aliases,
  iscsi_portal listen.ip, static_route destination/gateway, network_config
  gateways.
- **Per-resource timeout documentation** for long-running resources: vm (20m),
  certificate (20m ACME/CSR), pool (60m), systemdataset (20m), app (30m image
  pulls), replication (30m initial sync). Default `timeouts.Block` confirmed on
  all 62 resources.
- **filesystem_acl 404 handling**: Read and Delete handlers now use
  `client.IsNotFound(err)` to gracefully handle the case where the target
  filesystem path no longer exists, matching the pattern used by the other 45
  resources in the provider.
- **Acceptance test suite expanded from 30 → 156 functions across 58 files**
  (`_basic` + `_disappears` + `_update` triad for every CRUD
  resource, `_basic` + `_update` for singleton configs). The `_disappears`
  pattern calls the TrueNAS API directly via a test-only `testAccClient()`
  helper to out-of-band delete the resource, then asserts
  `ExpectNonEmptyPlan: true` so the provider is verified to detect and
  recover from external drift — the standard Terraform import-and-refresh
  pattern. All acceptance tests still gated on `TF_ACC=1` and run against
  the test VM only.
- **Test suite expanded from 221 → 1079 passing unit tests** (all automated via
  CI `go test -race -coverprofile`). Adds comprehensive httptest-mocked
  CRUD coverage for 42 client files (happy path, 404 → `IsNotFound`, 422/500 →
  `APIError`, invalid JSON, request-body marshaling, URL escaping, job-polling
  paths for async endpoints like certificate / systemdataset / app). Adds
  resource-layer `mapResponseToModel` fixture tests + schema validation tests
  across 30 resources. **Client package coverage: 87.8%**; validators 77.4%;
  plan modifiers 77.8%. All tests pass with `-race`.
- Dataset resource: fix `share_type` drift where SCALE 25.10 returns "GENERIC"
  on read regardless of create-time preset. `share_type` is now treated as a
  create-time preset and user intent is preserved across reads.
- Zvol resource: populate `volsize` and `volblocksize` on read by adding
  `DatasetResponse.Volsize` / `Volblocksize` fields and corresponding
  `GetVolsize()` / `GetVolblocksize()` helpers. Fixes post-import drift where
  both attributes appeared empty.
- Acceptance test fixes: real ed25519 keypair for `truenas_keychain_credential`
  (libcrypto rejects all-zero synthetic keys), `crypto/rand` UUID generator
  for `truenas_nvmet_host` NQN format, GRAPHITE-only attributes for
  `truenas_reporting_exporter`, `group_create` added to
  `ImportStateVerifyIgnore` on `truenas_user`.
- New `truenas_cloudsync_credential` resource + data source managing the
  `/cloudsync/credentials` API. Supports 16 provider types (S3, B2,
  Azure Blob, GCS, Dropbox, FTP, SFTP, HTTP, Mega, OpenStack Swift,
  pCloud, WebDAV, Yandex, OneDrive, Google Drive, Backblaze B2). Unblocks
  fully-Terraform `truenas_cloud_sync` / `truenas_cloud_backup` workflows.
- Provider-level `insecure_skip_verify` attribute and
  `TRUENAS_INSECURE_SKIP_VERIFY` environment variable for self-signed
  test environments.
- `client.IsNotFound(err)` helper that unwraps `*APIError` via
  `errors.As` and detects both HTTP 404 and TrueNAS 422
  "does not exist" responses. Applied to 44 resource Delete handlers so
  `terraform destroy` no longer fails when a resource has been removed
  out-of-band.
- Acceptance test scaffold (`internal/provider/provider_test.go`) with
  `testAccPreCheck` and a minimal `TestAccProvider_Schema` case under
  `TF_ACC=1`.
- Repo polish: `.golangci.yml` (13 linters, zero findings), `Makefile`
  with standard HashiCorp targets, `.editorconfig`, and new CI
  jobs for `golangci-lint`, `govulncheck`, `gitleaks`, and
  `tfplugindocs validate`.

### Fixed

- **SCALE 25.10 compatibility**: `iscsi_portal` no longer sends `port`
  in listen entries on create/update — the TrueNAS 25.10 API rejects it
  as "Extra inputs are not permitted". The field is now Computed-only
  and marked deprecated.
- `iscsi_extent`: `path` and `disk` are now `Computed: true` so DISK-type
  extents no longer trigger "Provider produced inconsistent result
  after apply" errors.
- `filesystem_acl`: validator now accepts NFS4 ACL tags (`owner@`,
  `group@`, `everyone@`) alongside the POSIX1E tags.
- `system_info` data source: `uptime_seconds` decoded as `float64`
  (SCALE 25.10 returns a float, not an int).
- `truenas_app`: `version = "latest"` no longer drifts to the resolved
  concrete version in state; the concrete value is exposed via
  `human_version`.
- 44 resource Read handlers: replaced broken `err.(*client.APIError)`
  type assertions (which silently failed on wrapped errors) with
  `client.IsNotFound(err)`.
- Lint cleanup across the codebase: 9 misspellings, 1 regex
  simplification, 4 unused `ctx` parameters, 1 empty branch, 2
  unchecked `fmt.Sscanf` return values, and 36 goimports adjustments.
- Consolidated conflicting `tools.go` at repo root into the canonical
  `tools/tools.go`.

## [0.4.0] - 2026-04-12

### Added

- 25 new resources: `vm`, `vm_device`, `app`, `catalog`, `privilege`,
  `kerberos_realm`, `kerberos_keytab`, `directoryservices`, `pool`,
  `network_interface`, `systemdataset`, `nvmet_global`, `nvmet_host`,
  `nvmet_subsys`, `nvmet_port`, `nvmet_namespace`, `nvmet_host_subsys`,
  `nvmet_port_subsys`, `vmware`, `cloud_backup`, `reporting_exporter`,
  `iscsi_auth`, `kmip_config`, `alertclasses`, `filesystem_acl_template`.
- 15 new data sources: `vm`, `vms`, `privilege`, `kerberos_realm`, `app`,
  `apps`, `catalog`, `directoryservices`, `systemdataset`,
  `network_interface`, `share_nfs`, `share_smb`, `cronjob`, `datasets`,
  `pools`.
- Field validators across all 61 resources (range, regex, enum,
  length).
- HTTP client retry/backoff with exponential jitter (max 5 attempts) and
  honoring of the `Retry-After` header on 429/503 responses.
- Per-resource `timeouts` block (create/read/update/delete).
- Generated documentation via `terraform-plugin-docs`.
- Runnable examples per resource in `examples/`.

### Fixed

- `filesystem_acl_template`: strip server-added `who: null` fields so
  round-trips match the user plan.
- `reporting_exporter`, `cloud_backup`: preserve only the user-supplied
  attributes subset; server-side defaults no longer trigger spurious drift.
- `vm_device`: filter server-added attribute keys (e.g. DISPLAY's
  `web_port`) to match the plan.
- `alert_service`: on SCALE 25.10, `type` is now embedded inside
  `attributes` (polymorphic discriminator schema); the top-level `type`
  field introduced in 25.04 has been reverted.
- `dataset`: read `comments` from `user_properties.comments` on SCALE
  25.10 (top-level `comments` field is now always null).

### Changed

- Split `internal/client/client.go` into per-domain files for
  maintainability. `client.go` now contains only the base HTTP client,
  retry/backoff helpers, `APIError`, `Job`/`WaitForJob`, and shared
  common types (`PropertyValue`, `PropertyRawVal`, `Schedule`).

## [0.3.0] - 2026-04-11

### Added

- 36 resources + 9 data sources covering the initial expansion wave:
  iSCSI target/portal/extent/initiator/target-extent, CronJob,
  AlertService, Replication, Zvol, User, Group, Tunable, CloudSync,
  RsyncTask, StaticRoute, NetworkConfiguration, InitScript, ScrubTask,
  FilesystemACL, Service, FTP/NFS/SMB/SNMP/SSH/UPS/Mail configs,
  ACME DNS authenticator, API key, Certificate, Keychain credential.
- Per-domain client files co-located with their resource definitions.
- Round-trip test coverage for every resource's Create→Read path.

### Fixed

- `dataset` now round-trips `comments`, `quota`, `refquota`, `sync`,
  `snapdir`, `copies`, `readonly`, and `recordsize` via the
  `PropertyValue`/`PropertyRawVal` indirection the SCALE API uses.

## [0.1.0] - 2026-04-11

### Added

- Initial provider implementation supporting TrueNAS SCALE 24.04+.
- API key authentication via `api_key` argument or `TRUENAS_API_KEY`
  environment variable.
- Base URL configuration via `url` argument or `TRUENAS_URL` environment
  variable.
- Built on `terraform-plugin-framework` v1.15.0.
- Initial resources: `dataset`, `share_nfs`, `share_smb`, `snapshot_task`,
  `replication`, `iscsi_portal`, `iscsi_initiator`, `iscsi_extent`,
  `iscsi_target`, `cronjob`, `alert_service`.
- Initial data sources: `dataset`, `pool`, `system_info`.

[Unreleased]: https://github.com/PjSalty/terraform-provider-truenas/compare/v1.0.0...main
[1.0.0]: https://github.com/PjSalty/terraform-provider-truenas/releases/tag/v1.0.0
[0.4.0]: https://github.com/PjSalty/terraform-provider-truenas/releases/tag/v0.4.0
[0.3.0]: https://github.com/PjSalty/terraform-provider-truenas/releases/tag/v0.3.0
[0.1.0]: https://github.com/PjSalty/terraform-provider-truenas/releases/tag/v0.1.0
