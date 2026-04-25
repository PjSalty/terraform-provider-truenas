---
page_title: "truenas_catalog Resource - terraform-provider-truenas"
subcategory: "Applications"
description: |-
  Manages the TrueNAS SCALE application catalog. This is a singleton resource in SCALE 25.04+ — only `preferred_trains` is user-tunable. Optionally triggers a catalog sync on create.
---

# truenas_catalog (Resource)

Manages the TrueNAS SCALE application catalog. This is a singleton resource in SCALE 25.04+ — only `preferred_trains` is user-tunable. Optionally triggers a catalog sync on create.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_catalog" "example" {
  label            = "custom"
  repository       = "https://github.com/example/catalog.git"
  branch           = "main"
  preferred_trains = ["stable"]
}
```

## Argument Reference

The following arguments are supported:

* `preferred_trains` - (Optional) Trains to prefer when searching for app versions (e.g. ['stable'], ['stable', 'community']).
* `sync_on_create` - (Optional) If true, trigger a catalog sync job after updating preferred_trains on create (default false).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_catalog` resource.
* `label` - The catalog label (always 'TRUENAS' in SCALE 25.04+).
* `location` - Local filesystem path where the catalog is checked out.

## Import

The `truenas_catalog` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_catalog.example custom
```
