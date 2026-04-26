---
page_title: "truenas_iscsi_extent Resource - terraform-provider-truenas"
subcategory: "iSCSI"
description: |-
  Manages an iSCSI extent on TrueNAS SCALE.
---

# truenas_iscsi_extent (Resource)

Manages an iSCSI extent on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_zvol" "iscsi_lun" {
  pool    = "tank"
  name    = "vols/iscsi-lun1"
  volsize = 10737418240 # 10 GiB
  sparse  = true
}

resource "truenas_iscsi_extent" "example" {
  name      = "iscsi-lun1"
  type      = "DISK"
  disk      = "zvol/tank/vols/iscsi-lun1"
  enabled   = true
  blocksize = 512
  rpm       = "SSD"
  comment   = "Example iSCSI LUN"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The extent name.
* `type` - (Required) The extent type (DISK or FILE). Valid values: `DISK`, `FILE`.
* `disk` - (Optional) The zvol path for DISK type extents.
* `path` - (Optional) The file path for FILE type extents. For DISK type extents, the API computes this from `disk` — leave unset.
* `filesize` - (Optional) The file size in bytes for FILE type extents.
* `blocksize` - (Optional) Block size in bytes (512 or 4096). Default: `512`.
* `rpm` - (Optional) Reported RPM (SSD, 5400, 7200, 10000, 15000). Default: `SSD`.
* `enabled` - (Optional) Whether the extent is enabled. Default: `true`.
* `comment` - (Optional) A comment for the extent.
* `readonly` - (Optional) Whether the extent is read-only. Default: `false`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_iscsi_extent` resource.

## Import

The `truenas_iscsi_extent` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_iscsi_extent.example 1
```
