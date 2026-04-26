---
page_title: "Architecture - TrueNAS Provider"
subcategory: "Guides"
description: |-
  An overview of how the TrueNAS Terraform provider is organized internally.
---

# Provider architecture

This page explains how `terraform-provider-truenas` is structured internally,
so contributors and advanced users understand where the code lives and how
the pieces interact. If you're only here to use the provider, skip this page
and read [Getting Started](getting-started.md) instead.

## High-level data flow

```
┌────────────────┐      ┌────────────────┐      ┌──────────────────┐
│   Terraform    │ ───▶ │     Provider   │ ───▶ │   REST client    │
│      core      │      │  (framework)   │      │   (httptest in   │
│                │ ◀─── │                │ ◀─── │   tests, real    │
└────────────────┘      └────────────────┘      │   in production) │
                                                └──────────────────┘
                                                          │
                                                          ▼
                                                ┌──────────────────┐
                                                │  TrueNAS SCALE   │
                                                │  REST /api/v2.0  │
                                                └──────────────────┘
```

Terraform core dispatches resource operations (`Create` / `Read` / `Update`
/ `Delete` / `ImportState`) into the provider binary via gRPC. The provider
binary is a thin shim over the terraform-plugin-framework runtime. Each
resource implementation decodes plan / state / config values, calls the
REST client to mutate TrueNAS, and writes results back into state.

## Package layout

```
internal/
├── client/          TrueNAS REST client, per-domain file per API family
├── datasources/     Terraform data sources (read-only)
├── resources/       Terraform resources (full CRUD + ImportState)
├── provider/        Provider registration and acceptance tests
├── validators/      Reusable attribute validators
├── planmodifiers/   Reusable plan modifiers
├── flex/            Framework <-> Go type conversion helpers
├── fwresource/      Framework resource base helpers (Configure boilerplate)
├── acctest/         Shared acceptance test helpers
└── sweep/           Acceptance test sweeper infrastructure
```

### `internal/client`

Implements the TrueNAS REST API as a typed Go client. One file per API
domain (`dataset.go`, `share_nfs.go`, `iscsi_target.go`, etc.) — this keeps
diffs small and matches the layout of `internal/resources`.

Key types:

- `Client` — holds base URL, API key, HTTP client, and retry policy.
- `APIError` — wraps a non-2xx response, including status code, body, and
  parsed error message. Use `errors.As(err, &apiErr)` to access fields.
- `IsNotFound(err)` — returns true for HTTP 404 *or* TrueNAS's "422 does not
  exist" responses (the API isn't consistent about which it returns). Every
  resource Delete handler uses this to make `terraform destroy` idempotent.

The client retries idempotent methods (GET / PUT / DELETE / HEAD) on
transient failures (429 / 5xx / transport errors) with exponential backoff
and `Retry-After` honoring. POST is not retried on HTTP errors — only on
transport errors where we know the request body never reached the server —
to avoid duplicate creates.

### `internal/resources`

One file per Terraform resource. Each resource implements:

- `resource.Resource` — `Metadata`, `Schema`, `Configure`, `Create`, `Read`,
  `Update`, `Delete`.
- `resource.ResourceWithImportState` — `ImportState` (numeric ID passthrough
  for most, compound IDs for a few).
- `resource.ResourceWithModifyPlan` (optional) — cross-attribute validation
  at plan time.

Every resource uses a `timeouts.Block` for per-resource Create/Read/Update/
Delete timeouts and defaults. Credential-bearing attributes are marked
`Sensitive: true`.

### `internal/datasources`

Read-only counterparts to resources. One data source per resource typically,
plus singletons for global config (`system_info`, `network_config`, etc.).

### `internal/provider`

- `provider.go` — provider registration, `Resources()` / `DataSources()`
  registries.
- `acc_*_test.go` — acceptance tests (one file per resource), guarded by
  `TF_ACC=1`.
- `sweeper_test.go` — sweeper registrations that clean up abandoned test
  fixtures via `go test -sweep=all`.

## Schema and state flow

```
User HCL        Plan            Config          State
  │              │                │                │
  ▼              ▼                ▼                ▼
┌─────────────────────────────────────────────────────────┐
│                terraform-plugin-framework               │
│                                                         │
│  Schema → Plan decode → Validate → Apply → State write  │
└─────────────────────────────────────────────────────────┘
          │                 │            │
          │                 │            │
          ▼                 ▼            ▼
    Schema validators   ModifyPlan   CRUD handlers
    (per-attribute)     (cross-attr) (client calls)
```

1. Terraform core decodes the user's HCL into a raw plan value.
2. The framework runs per-attribute validators (`validators.ZFSPath`, etc.)
   and plan modifiers (`RequiresReplaceIfChanged`, etc.).
3. Resources implementing `ModifyPlan` get a chance to add cross-attribute
   diagnostics before apply.
4. On apply, Terraform calls the appropriate CRUD method. The handler
   decodes plan/state/config, calls the REST client, and writes results
   back into state.

## Error handling

All provider errors surface as Terraform diagnostics. The convention:

- **API errors** → `resp.Diagnostics.AddError(summary, fmt.Sprintf("... %s", err))`.
- **Attribute-level errors** → `resp.Diagnostics.AddAttributeError(path, summary, detail)`.
- **404 on Read** → `resp.State.RemoveResource(ctx)` without adding an error.
  This is how Terraform discovers out-of-band deletion and plans a recreate.
- **404 on Delete** → silently return success. Idempotent destroy.

The client's `IsNotFound(err)` wraps `errors.As` for `*APIError` and checks
both HTTP 404 and TrueNAS's 422 "does not exist" responses.

## Retry and backoff

Client-level retries are handled in `client.doRequest`:

- Idempotent HTTP methods retry on 429/500/502/503/504 and transport errors.
- POST retries only on transport errors that the server never saw.
- 429/503 responses honor `Retry-After` (either seconds or HTTP-date).
- Backoff is exponential with up to 25% jitter, capped at `policy.MaxDelay`.
- Context cancellation aborts the retry loop immediately.

Resource-level retries are handled by `timeouts.Block` — long-running
operations like `truenas_pool` create or `truenas_certificate` ACME issuance
extend the default timeout via per-resource defaults.

## Sensitive attributes

The provider marks every credential-bearing attribute as `Sensitive: true`
so Terraform redacts it from plan output, state files, and the CLI UI.
Sensitive attributes are audited via the `internal/resources/*_test.go`
schema tests. Lint policy: any attribute named `password`, `secret`,
`api_key`, `token`, `passphrase`, `private_key`, `dhchap_key`,
`credentials_json`, or similar **must** be `Sensitive: true`.

## Testing strategy

| Layer | Harness | What it verifies |
|-------|---------|------------------|
| Unit | `net/http/httptest` | REST client + resource CRUD handlers against mocked responses |
| Fuzz | Go native fuzzing | Parser/validator/serializer never panic |
| Benchmark | `go test -bench` | Hot-path performance (map-to-model, client round-trip) |
| Acceptance | Real TrueNAS VM | End-to-end CRUD + import + drift detection against SCALE |

Coverage target is **100.0%** on every package. CI enforces this gate.

## Release pipeline

Tag `vX.Y.Z` on `main` to trigger goreleaser in CI. The release job:

1. Builds binaries for 14 platform targets (linux/darwin/windows/freebsd ×
   amd64/arm64/arm6/arm7/386).
2. Generates SBOMs via `syft`.
3. Signs the checksum file with GPG (`GPG_FINGERPRINT` secret).
4. Publishes a GitHub release with changelog, SBOMs, signatures, and
   `terraform-registry-manifest.json`.

See [`.goreleaser.yml`](../.goreleaser.yml) for the full config.

## Further reading

- [Getting Started](getting-started.md)
- [Importing Existing Resources](importing-existing.md)
- [Backup Strategy](backup-strategy.md)
- [Kubernetes Storage via democratic-csi](kubernetes-storage.md)
