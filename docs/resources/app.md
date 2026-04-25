---
page_title: "truenas_app Resource - terraform-provider-truenas"
subcategory: "Applications"
description: |-
  Manages a deployed application on TrueNAS SCALE (Docker/iX). Install is asynchronous — the provider waits for the underlying job to complete.
---

# truenas_app (Resource)

Manages a deployed application on TrueNAS SCALE (Docker/iX). Install is asynchronous — the provider waits for the underlying job to complete.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

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

## Argument Reference

The following arguments are supported:

* `app_name` - (Required) The app name. Must be lowercase alphanumeric with hyphens, starting with a letter (e.g. 'my-app'). Immutable after creation. Changing this attribute forces a new resource to be created.
* `catalog_app` - (Required) The catalog app slug to install (e.g. 'minio', 'plex'). Immutable after creation. Changing this attribute forces a new resource to be created.
* `train` - (Optional) The catalog train (e.g. 'stable', 'enterprise', 'community'). Immutable after creation. Default: `stable`. Changing this attribute forces a new resource to be created.
* `version` - (Optional) The app chart version to install (e.g. '1.2.3'). Defaults to 'latest'. Immutable after creation — use the TrueNAS upgrade workflow for in-place version changes. Default: `latest`. Changing this attribute forces a new resource to be created.
* `values` - (Optional) JSON-encoded values object passed to the app. Arbitrary chart configuration — the provider does not validate structure. Default: `{}`.
* `remove_images` - (Optional) On destroy, remove associated container images (default true).
* `remove_ix_volumes` - (Optional) On destroy, also remove ix-volumes (default false — DANGEROUS).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

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
