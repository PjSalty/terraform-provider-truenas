---
page_title: "truenas_iscsi_targetextent Resource - terraform-provider-truenas"
subcategory: "iSCSI"
description: |-
  Manages an iSCSI target-to-extent association on TrueNAS SCALE. This links an iSCSI target to an extent, completing the iSCSI stack.
---

# truenas_iscsi_targetextent (Resource)

Manages an iSCSI target-to-extent association on TrueNAS SCALE. This links an iSCSI target to an extent, completing the iSCSI stack.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_iscsi_targetextent" "example" {
  target = tonumber(truenas_iscsi_target.example.id)
  extent = tonumber(truenas_iscsi_extent.example.id)
  lunid  = 0
}

resource "truenas_iscsi_target" "example" {
  name = "iqn.2025-01.com.example:target1"
  mode = "ISCSI"
  groups {
    portal    = 1
    initiator = 1
  }
}

resource "truenas_iscsi_extent" "example" {
  name = "example-lun"
  type = "DISK"
  disk = "zvol/tank/example"
}
```

## Argument Reference

The following arguments are supported:

* `target` - (Required) The iSCSI target ID.
* `extent` - (Required) The iSCSI extent ID.
* `lunid` - (Optional) The LUN ID. If not set, it is auto-assigned by TrueNAS.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_iscsi_targetextent` resource.

## Import

The `truenas_iscsi_targetextent` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_iscsi_targetextent.example 1
```
