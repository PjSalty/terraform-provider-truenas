---
page_title: "Upgrading to v1.0 - TrueNAS Provider"
subcategory: "Guides"
description: |-
  Migration notes, stability guarantees, and schema-version policy for the v1.0 release of the TrueNAS Terraform provider.
---

# Upgrading to v1.0

This guide covers everything you need to know about moving from the v0.x releases of `terraform-provider-truenas` to v1.0 and what the v1.0 contract guarantees going forward.

## TL;DR

- **No breaking changes from the last v0.x release.** v1.0 is the same code that shipped in the final v0.x with the version bumped — no provider blocks, resource attributes, or data source shapes change.
- **Minimum Terraform version: 1.5**. Stay on a supported Terraform release.
- **Minimum TrueNAS SCALE version: 24.04**. Older versions still *may* work but are no longer tested in CI.
- **No configuration changes required.** Bump the version constraint, run `terraform init -upgrade`, and run `terraform plan`. A clean plan should show no drift.

## Why v1.0?

The v0.x series was the shakedown phase: the resource surface, schema shapes, and provider configuration stabilized across dozens of releases. v1.0 formalizes that surface as a stable API covered by the stability policy below. In practice this means:

1. **Downstream users can pin a major version** (`~> 1.0`) and expect non-breaking minor/patch updates for the lifetime of v1.
2. **Breaking schema changes will be gated behind v2.0** and documented in a follow-up upgrade guide.
3. **Fixes and new resources land in v1.x** without requiring coordinated user-side migrations.

## Stability Guarantees (v1.x)

The following are covered by the v1.x stability promise. Any change that violates these guarantees counts as a breaking change and will ship in v2.0.

| Surface | Guarantee |
|---------|-----------|
| Resource type names (`truenas_*`) | Stable — no renames, no removals without a deprecation cycle. |
| Data source type names | Stable. |
| Required attributes | An attribute marked `Required` in v1.x stays `Required`. It will not become `Optional` in a way that changes default semantics. |
| Optional attributes with non-null defaults | The default value will not change. If TrueNAS itself changes a default, the provider will pin the previous value or introduce a new attribute. |
| `Computed` attribute shapes (types, nested structure) | Stable. New fields MAY be added; existing fields will not disappear or change type. |
| `id` format for each resource | Stable — existing state files continue to import cleanly. |
| Plan-modifier semantics (`RequiresReplace`, `UseStateForUnknown`, etc.) | Stable. An attribute marked `RequiresReplace` in v1.x will not silently become in-place updatable (though the reverse — relaxing a constraint — may happen in a minor release). |
| Provider block attributes (`url`, `api_key`, `insecure_skip_verify`) | Stable. New optional fields may be added. |

### What is NOT covered

- **Warning/error message wording.** The provider may improve diagnostic messages without versioning the change.
- **Internal client package (`internal/client`).** Not part of the public API. Do not import it from outside the provider.
- **`tflog` output.** Trace/debug log content and formatting are subject to change.
- **Underlying TrueNAS API quirks.** If TrueNAS SCALE changes an endpoint shape in a point release, the provider will adapt. We document any user-visible impact in the release notes.

## Deprecation Policy

When a resource, data source, or attribute needs to be removed in a future major (v2.0), the deprecation cycle is:

1. **Minor release (e.g., v1.5)**: The field is marked `DeprecationMessage` in the schema. Terraform plans show a warning but the behavior is unchanged.
2. **One or more minor releases later**: The deprecation continues to warn. The `CHANGELOG.md` highlights the upcoming removal.
3. **Next major (v2.0)**: The field is removed. A migration path is documented in `docs/guides/upgrade-to-v2.md` (shipped alongside v2.0.0).

No field will be removed in a minor release without a prior deprecation warning in an earlier minor release.

## Schema-Version Migrations

Some resources evolve their internal state shape even when the user-facing HCL stays the same — for example, when TrueNAS adds a new field that the provider now surfaces as `Computed`. Terraform supports this via **schema versions** and **state upgraders**: a resource declares `SchemaVersion: N`, and for each previous version the resource implements an upgrade function that migrates old state into the new shape.

### When you will see this

You should not have to care about schema versions in day-to-day use. When you upgrade the provider:

1. Terraform reads the stored schema version from your state file.
2. If it is older than the current `SchemaVersion`, Terraform walks each `StateUpgrader` in order and writes the upgraded state back.
3. Your next `terraform plan` runs against the upgraded state and, ideally, shows no drift.

The upgrade is automatic and non-destructive. State upgraders only transform in-memory state — they never make API calls to TrueNAS. If you see an error that mentions `state upgrader` or `schema version mismatch`, file a bug.

### Example: `truenas_cronjob`

The cron job resource is the canonical example of a schema-version migration in this provider. v0 represented `schedule` as a flat map of strings; v1 adopted a nested block for better validation. The resource declares:

```go
func (r *CronJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    // Current (v1) schema definition...
    resp.Schema.Version = 1
}

func (r *CronJobResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
    v0 := cronjobSchemaV0(ctx)
    return map[int64]resource.StateUpgrader{
        0: {
            PriorSchema: &v0,
            StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
                // transform v0 state into v1 shape
            },
        },
    }
}
```

See `internal/resources/cronjob.go` for the full implementation. This pattern is the template for any future schema migrations in v1.x; contributors adding new `SchemaVersion` bumps should follow it.

### What NOT to do

- Do not hand-edit `terraform.tfstate`. If your state file is on an older schema, run `terraform plan` and let the provider upgrade it.
- Do not delete the `schema_version` field from state. It is how the upgrade chain is driven.

## Alpha / Beta Resource Graduation

Some resources ship marked `Alpha` or `Beta` in their description. These are experimental surfaces that may still see breaking schema changes within v1.x. They are:

- **Alpha**: Resource exists and works, but the schema (attribute names, required/optional flags) is not yet frozen. Safe to experiment with, not safe to depend on.
- **Beta**: Schema is almost frozen but the maintainers want one or two releases of production exposure before declaring it stable. Small non-breaking additions are possible; removals or renames will be avoided.
- **Stable (default)**: Covered by the v1.x stability promise above.

### Graduation path

An Alpha → Beta → Stable graduation happens in a minor release. The resource's description is updated to drop the Alpha/Beta label and the `CHANGELOG.md` notes the transition. Once stable, the resource is bound by the usual v1.x guarantees.

If an Alpha/Beta resource needs a breaking change during v1.x, it happens in a minor release *with the Alpha/Beta label still attached* and is called out loudly in the release notes. Stable resources never break in a minor release.

### Current Alpha/Beta resources

At the time of v1.0 GA, the following resources carry pre-stable labels:

- None. All resources that shipped in v0.x are stable in v1.0.

Future experimental resources added during v1.x will be clearly labeled in their `Description` field and on the registry documentation page.

## Upgrade Procedure

1. Pin the version in your provider block:

    ```terraform
    terraform {
      required_providers {
        truenas = {
          source  = "registry.terraform.io/hashicorp/truenas"
          version = "~> 1.0"
        }
      }
    }
    ```

2. Run `terraform init -upgrade`.

3. Run `terraform plan`. A clean upgrade from the final v0.x release produces an empty plan.

4. If the plan shows drift on attributes you did not touch, open an issue with the relevant plan output — that is a regression we want to fix in v1.0.x.

5. Run `terraform apply` to apply any expected drift (e.g., from a concurrent out-of-band change) and continue normal operations.

## Getting Help

- **Release notes**: `CHANGELOG.md` in the repository.
- **Stability questions**: Open a discussion on the GitHub repository.
- **Bug reports / regressions**: Open an issue with the failing plan output, Terraform version, provider version, and TrueNAS SCALE version.

## Looking Ahead

v1.0 freezes the current resource surface. v1.x releases will focus on:

- New resources covering the remaining TrueNAS SCALE REST surface.
- New data sources.
- Retry/error-handling polish.
- Documentation improvements.
- Performance optimizations (batch reads, caching).

Breaking changes will accumulate in a v2 branch and ship when the cost of migration is outweighed by the benefit. This guide will be joined by `docs/guides/upgrade-to-v2.md` when that happens.
