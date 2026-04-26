---
page_title: "truenas_iscsi_target Resource - terraform-provider-truenas"
subcategory: "iSCSI"
description: |-
  Manages an iSCSI target on TrueNAS SCALE.
---

# truenas_iscsi_target (Resource)

Manages an iSCSI target on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_iscsi_portal" "t" {
  listen {
    ip   = "0.0.0.0"
    port = 3260
  }
}

resource "truenas_iscsi_initiator" "t" {
  comment = "Allow all"
}

resource "truenas_iscsi_target" "example" {
  name  = "iqn.2025-01.com.example:target1"
  alias = "example-target"
  mode  = "ISCSI"

  groups {
    portal    = truenas_iscsi_portal.t.tag
    initiator = tonumber(truenas_iscsi_initiator.t.id)
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The iSCSI target name (IQN suffix).
* `alias` - (Optional) An optional alias for the target.
* `mode` - (Optional) The target mode (ISCSI, FC, BOTH). Valid values: `ISCSI`, `FC`, `BOTH`. Default: `ISCSI`.
* `groups` - (Optional) Target groups linking portals and initiators.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_iscsi_target` resource.

## Import

The `truenas_iscsi_target` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_iscsi_target.example 1
```
