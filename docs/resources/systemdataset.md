---
page_title: "truenas_systemdataset Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages the TrueNAS system dataset pool assignment. This is a singleton — only one instance of this resource can exist per TrueNAS system.
---

# truenas_systemdataset (Resource)

Manages the TrueNAS system dataset pool assignment. This is a singleton — only one instance of this resource can exist per TrueNAS system.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
data "truenas_pool" "tank" {
  name = "tank"
}

resource "truenas_systemdataset" "this" {
  pool = data.truenas_pool.tank.name
}
```

## Argument Reference

The following arguments are supported:

* `pool` - (Required) The name of the pool hosting the system dataset. Set to an empty string to let TrueNAS select the default (boot pool).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_systemdataset` resource.
* `pool_set` - Whether a pool has been explicitly set for the system dataset.
* `uuid` - UUID of the system dataset.
* `basename` - Base name of the system dataset.
* `path` - Filesystem path to the system dataset.

## Import

The `truenas_systemdataset` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_systemdataset.this singleton
```
