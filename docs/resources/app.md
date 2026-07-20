---
page_title: "truenas_app Resource - terraform-provider-truenas"
subcategory: "Applications"
description: |-
  Manages a deployed application on TrueNAS SCALE (Docker/iX). Install is asynchronous, the provider waits for the underlying job to complete.
---

# truenas_app (Resource)

Manages a deployed application on TrueNAS SCALE (Docker/iX). Install is asynchronous, the provider waits for the underlying job to complete.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

Two install modes are supported, selected by exactly one of `catalog_app`
(a catalog install) or `custom_compose` (a custom Docker Compose app).

## Example Usage

### Catalog app

```terraform
resource "truenas_app" "example" {
  app_name    = "plex"
  catalog_app = "plex"
  train       = "community"
  version     = "1.0.0"
  values = jsonencode({
    plex = {
      claimToken = ""
    }
  })
}
```

### Custom Docker Compose app

```terraform
resource "truenas_app" "sleeper" {
  app_name       = "sleeper"
  custom_compose = <<-EOT
    services:
      app:
        image: busybox:1.36
        command: ["sleep", "infinity"]
        restart: unless-stopped
  EOT
}
```

Compose content edits (image bumps, environment changes) apply in place.
Converting an existing app between catalog and custom forces replacement.

#### Compose drift semantics

The middleware stores the parsed compose document, not your YAML string, so
comparison is semantic: reformatting your config (indentation, comments,
key order, quoting, block vs flow style) of a structurally unchanged
compose keeps the stored state value and plans no diff at all. On every
refresh the provider fetches the server's stored compose and checks it
structurally against your string; real drift (a changed value, an added
service) surfaces as a normal plan diff and the next apply repairs it in
place.

The middleware parses with YAML 1.1 rules, so unquoted `yes`/`no`/`on`/`off`
values become booleans server-side. The provider tolerates that skew for
VALUES (exactly those spellings in lower, Title, and UPPER case), but
quoting such values (`"yes"`) keeps intent unambiguous and is recommended.

Mapping KEYS compare literally, no numeric or boolean tolerance applies to
them. Quote any key YAML could type-convert (bare numbers, `on`, `off`,
`yes`, `no`), otherwise it will read back differently and drift forever.

#### Secrets in compose files

Compose files often carry credentials in `environment` blocks. The
middleware wraps the compose fields as `Secret` server-side, but the string
you write lives in the Terraform state and plan output like any other
attribute. Put secrets in referenced env files on the NAS (`env_file`) if
you do not want them in state.

#### Known sharp edges

* 25.04.x: the middleware's re-serialization can unquote numeric-looking
  strings such as `"8E1"` and break the compose (NAS-136877, fixed in
  25.10).
* `app.update` on a STOPPED app writes the config but does not deploy it
  until the next start; the apply succeeds, deployment is deferred.
* An app whose container healthcheck never passes sits in
  STARTING/DEPLOYING by design.
* 25.10.1+ validates via `docker compose config` server-side with a 30s
  timeout; very large compose documents can hit it.

## Argument Reference

The following arguments are supported:

* `app_name` - (Required) The app name. Must be lowercase alphanumeric with hyphens, starting with a letter (e.g. 'my-app'). Immutable after creation. Changing this attribute forces a new resource to be created.
* `catalog_app` - (Optional) The catalog app slug to install (e.g. 'minio', 'plex'). Exactly one of `catalog_app` or `custom_compose` must be set. Immutable after creation. Changing this attribute forces a new resource to be created.
* `custom_compose` - (Optional) Raw Docker Compose YAML for a custom app install. Exactly one of `catalog_app` or `custom_compose` must be set, and it conflicts with the catalog-only knobs (`train`, `version`, `values`). Content edits apply in place; converting between catalog and custom forces replacement. See the drift semantics above.
* `train` - (Optional) The catalog train (e.g. 'stable', 'enterprise', 'community'). Immutable after creation. Default: `stable`. Changing this attribute forces a new resource to be created.
* `version` - (Optional) The app chart version to install (e.g. '1.2.3'). Defaults to 'latest'. Immutable after creation, use the TrueNAS upgrade workflow for in-place version changes. Default: `latest`. Changing this attribute forces a new resource to be created.
* `values` - (Optional) JSON-encoded values object passed to the app. Arbitrary chart configuration, the provider does not validate structure. Default: `{}`.
* `remove_images` - (Optional) On destroy, remove associated container images (default true).
* `remove_ix_volumes` - (Optional) On destroy, also remove ix-volumes (default false, DANGEROUS).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

-> Custom apps are destroyed with `force_remove_custom_app`, so a broken or
missing compose file on the NAS cannot wedge a destroy.

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_app` resource.
* `state` - The app runtime state (RUNNING, STOPPED, DEPLOYING, CRASHED).
* `upgrade_available` - Whether a newer chart version is available.
* `human_version` - Human-readable app version string.

## Import

The `truenas_app` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_app.example plex
```

For custom compose apps the original YAML string cannot be recovered (the
middleware stores the parsed document), so the first refresh after import
fills `custom_compose` with a canonical dump of the server's compose. A
config that matches the server structurally plans clean immediately, the
semantic comparison ignores formatting differences.
