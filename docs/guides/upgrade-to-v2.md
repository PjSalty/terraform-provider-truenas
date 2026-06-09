---
page_title: "Upgrading to v2.0 - TrueNAS Provider"
subcategory: "Guides"
description: |-
  Transport cutover, TrueNAS version requirement, and rollback path for the v2.0 release of the TrueNAS Terraform provider.
---

# Upgrading to v2.0

This guide covers everything you need to know about moving from the v1.x releases of `terraform-provider-truenas` to v2.0.

## TL;DR

- **The default transport flips from REST to JSON-RPC 2.0 over WebSocket.** This is the *only* user-visible change.
- **No schema changes.** Existing Terraform configurations and state files keep working — every resource and data source ID, attribute, and import path is identical to v1.x.
- **Recommended upgrade flow:** bump the version constraint, run `terraform init -upgrade`, then `terraform plan`. A clean plan shows no drift.

## Why v2.0?

TrueNAS SCALE 25.04 introduced JSON-RPC 2.0 over WebSocket at `/api/current` and surfaced a "deprecated REST API was used" alert on every call against the legacy `/api/v2.0` endpoints. iX has scheduled REST removal for SCALE 26.04. Continuing on the REST default would mean the provider stops working the day a homelab box upgrades to 26.04.

v2.0 swaps the default to WebSocket. The implementation has been in tree since v1.10.x as `transport = "websocket"`; v2.0 only flips the default — every resource and data source has been on the dual-transport path for at least a release.

## What changed

### Default transport

**REST is fully retired in v2.0.** The `internal/client/` package is gone; resource I/O flows exclusively over JSON-RPC over WebSocket at `/api/current`. There is no REST fallback. Operators on TrueNAS SCALE versions older than 25.04 (which is when WebSocket landed) must stay on the v1.x provider line until they upgrade their TrueNAS.

### TrueNAS version requirement

| Provider | Minimum SCALE |
| --- | --- |
| v1.x | 24.04 (REST only) |
| v2.0 (WebSocket only) | 25.04 |

If the provider Configure step fails with a websocket dial error against a SCALE 24.10 (or older) host, that's the version mismatch. Upgrade SCALE to 25.04 or newer, or stay on the v1.x provider line.

### REST API deprecation timeline

iX's published timeline for the REST API at `/api/v2.0`:

- **SCALE 25.04** — deprecated; alert fires on every call.
- **SCALE 26.04** — removed entirely.

This provider's timeline:

- **v1.x** — REST is the default. WebSocket is opt-in alpha via `transport = "websocket"`.
- **v2.0** — WebSocket only. The REST client code has been deleted. Provider requires SCALE 25.04+ unconditionally.

## Upgrade procedure

1. Bump the version constraint in your `required_providers` block:

    ```hcl
    terraform {
      required_providers {
        truenas = {
          source  = "PjSalty/truenas"
          version = "~> 2.0"
        }
      }
    }
    ```

2. Run `terraform init -upgrade`. The lockfile (`.terraform.lock.hcl`) updates with the v2.0 provider hashes.

3. Run `terraform plan`. A clean plan should show **no resource changes** — only the provider version line moves.

4. If the plan is clean, run `terraform apply`. The first apply re-Configures the provider and opens the WebSocket connection. After that it behaves identically to v1.x.

If you see any resource diff that doesn't match a real intent change, see "Rollback" below.

## Rollback

If something goes wrong after the upgrade — unexpected diffs, dial errors, or any other regression — fall back to REST instantly without touching state:

```hcl
provider "truenas" {
  url     = "..."
  api_key = "..."

  # Pin to the v1.x transport to bypass any v2.0 behavior.
  transport = "rest"
}
```

Or via environment:

```sh
export TRUENAS_TRANSPORT=rest
```

Either approach selects the REST client, which is the same code path as v1.10.x. Then re-run `terraform plan`. If that produces a clean plan, the issue is WebSocket-specific and worth filing as an issue with the offending resource type and `terraform plan` output.

The REST option is supported through the entire v2.x line; v2.1 is the first release that drops it.

## Behavioral parity

Every resource and data source has been verified to produce the same `terraform plan` output under both transports against the test VM. The schemas, IDs, and import paths are identical. If you find a case where REST and WebSocket disagree on a resource shape, that's a bug — please file an issue.

Two areas where the transports differ in *implementation* but not user-visible behavior:

1. **Long-running operations** (pool create/export, certificate create/update/delete, app install/upgrade/uninstall, system_dataset move): both transports poll `core.get_jobs` until terminal state. The WebSocket implementation lets the connection survive multiple jobs without re-handshaking, so a long apply with many resources is somewhat faster, but each individual operation has identical observable semantics.

2. **`*ByName` lookups** (e.g. `GetCertificateByName`, `GetServiceByName`): the REST client lists all entries and filters client-side. The WebSocket client uses server-side filtering on the `*.query` method, which is faster on hosts with many entries but produces identical results.

## Concurrency and rate-limit behavior

WebSocket multiplexes many in-flight calls over a single connection. The provider gates outgoing calls with a semaphore sized from the server's reported rate limit on connect, so `terraform apply -parallelism=N` continues to work without hitting `-32000 too many concurrent calls`.

The REST client's per-request HTTP retry envelope (jittered backoff on 5xx and `Retry-After`) carries forward into the WebSocket client's reconnect logic. If the server restarts mid-apply, idempotent in-flight calls (reads, PUT-style updates) retry transparently after reconnect; non-idempotent calls (creates, deletes) error fast with the connection-lost context so the operator can rerun.

## What does NOT change

- **Provider attributes**: `url`, `api_key`, `insecure_skip_verify`, `read_only`, `destroy_protection`, `request_timeout`, `transport` — all unchanged.
- **Resource and data source schemas**: every one of the 63 resources and their data sources keeps the same attributes, types, validators, plan modifiers, and import paths.
- **State file format**: existing state files load directly into v2.0. No `terraform state` migration is required.
- **Acceptance test coverage**: 100% line coverage gate held throughout the migration; the same test suite covers both transports.

## Stability guarantees (v2.x)

Same shape as v1.x: schema-stable across the major version, breaking changes gated behind v3.0. The transport-default flip in v2.0 was *not* a schema break — schemas are byte-identical with v1.10.2.

The only v2.x → v2.x change scheduled today is the v2.1 deletion of the `transport = "rest"` rollback path. That is documented as a deprecation in the v2.0 schema description; if you need REST past v2.x, pin to `~> 2.0` rather than `~> 2`.

## Reporting issues

If the upgrade surfaces anything unexpected:

1. Set `transport = "rest"` to confirm the REST path still works for your config. If it does, the issue is WebSocket-specific.
2. File an issue at https://github.com/PjSalty/terraform-provider-truenas/issues with the resource type, the unexpected diff or error, and the result of step 1.

The original WebSocket migration request is [issue #8](https://github.com/PjSalty/terraform-provider-truenas/issues/8) for context on the cutover.
