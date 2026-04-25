---
page_title: "truenas_share_smb Resource - terraform-provider-truenas"
subcategory: "Sharing"
description: |-
  Manages an SMB share on TrueNAS SCALE.
---

# truenas_share_smb (Resource)

Manages an SMB share on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_dataset" "smb_share" {
  pool = "tank"
  name = "smb_share"
}

resource "truenas_share_smb" "example" {
  path      = "/mnt/tank/smb_share"
  name      = "example"
  comment   = "Example SMB share"
  purpose   = "DEFAULT_SHARE"
  enabled   = true
  browsable = true
  home      = false
  read_only = false
  guestok   = false
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path to share (e.g., /mnt/tank/data).
* `name` - (Required) The share name visible to SMB clients.
* `comment` - (Optional) A comment describing the share.
* `browsable` - (Optional) Whether the share is browsable in network discovery. Default: `true`.
* `readonly` - (Optional) Whether the share is read-only. Default: `false`.
* `abe` - (Optional) Whether Access Based Share Enumeration is enabled. Default: `false`.
* `enabled` - (Optional) Whether the share is enabled. Default: `true`.
* `purpose` - (Optional) The share purpose preset. Valid values: `DEFAULT_SHARE`, `ENHANCED_TIMEMACHINE`, `LEGACY_SMB_WHITELIST`, `MULTI_PROTOCOL_NFS`, `MULTI_PROTOCOL_AFP`, `PRIVATE_DATASETS`, `NO_PRESET`, `TIMEMACHINE`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_share_smb` resource.

## Import

The `truenas_share_smb` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_share_smb.example 1
```
