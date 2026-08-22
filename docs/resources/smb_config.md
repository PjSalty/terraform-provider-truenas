---
page_title: "truenas_smb_config Resource - terraform-provider-truenas"
subcategory: "Sharing"
description: |-
  Manages the SMB service configuration on TrueNAS SCALE. This is a singleton resource, only one instance can exist.
---

# truenas_smb_config (Resource)

Manages the SMB service configuration on TrueNAS SCALE. This is a singleton resource, only one instance can exist.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_smb_config" "this" {
  netbiosname      = "truenas"
  workgroup        = "WORKGROUP"
  description      = "TrueNAS SCALE SMB server"
  minimum_protocol = "SMB2"
  unixcharset      = "UTF-8"
  enable_smb1      = false
  aapl_extensions  = false
  guest            = "nobody"
  filemask         = "0775"
  dirmask          = "0775"
}
```

## Argument Reference

The following arguments are supported:

* `netbiosname` - (Optional) NetBIOS name of the server. Default: `truenas`.
* `workgroup` - (Optional) Windows workgroup name. Default: `WORKGROUP`.
* `description` - (Optional) Server description. Default: `TrueNAS Server`.
* `search_protocols` - (Optional) Extra search protocols the SMB server
  answers. Currently only `SPOTLIGHT` (macOS Spotlight). **Requires TrueNAS
  26.0 or newer**; it is only sent when set, and asking for it against an
  older server is a clear error rather than a middleware validation failure.
* `minimum_protocol` - (Optional) Minimum SMB protocol version the server will
  negotiate: `SMB1`, `SMB2`, or `SMB3`. `SMB3` requires TrueNAS 26.0 or newer.
* `enable_smb1` - (Optional, **deprecated**) Enable SMB1 protocol support. Use
  `minimum_protocol` instead. The two are kept in sync automatically and cannot
  both be set.

  | `enable_smb1` | `minimum_protocol` |
  | --- | --- |
  | `true` | `SMB1` |
  | `false` | `SMB2` |
  | (not expressible) | `SMB3` |

  TrueNAS 26.0 replaced `enable_smb1` with `minimum_protocol`. The provider
  detects which the server speaks and sends the right one, so `minimum_protocol`
  works on every supported version.
* `unixcharset` - (Optional) UNIX character set. Default: `UTF-8`.
* `aapl_extensions` - (Optional) Enable Apple SMB2/3 protocol extensions. Default: `false`.
* `guest` - (Optional) Guest account for unauthenticated access. Default: `nobody`.
* `filemask` - (Optional) File creation mask. Default: `DEFAULT`.
* `dirmask` - (Optional) Directory creation mask. Default: `DEFAULT`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_smb_config` resource.

## Import

The `truenas_smb_config` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_smb_config.this singleton
```
