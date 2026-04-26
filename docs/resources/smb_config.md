---
page_title: "truenas_smb_config Resource - terraform-provider-truenas"
subcategory: "Sharing"
description: |-
  Manages the SMB service configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.
---

# truenas_smb_config (Resource)

Manages the SMB service configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_smb_config" "this" {
  netbiosname = "truenas"
  workgroup   = "WORKGROUP"
  description = "TrueNAS SCALE SMB server"
  enable_smb1 = false
  unixcharset = "UTF-8"
  loglevel    = "MINIMUM"
  syslog      = false
  localmaster = true
  guest       = "nobody"
  filemask    = "0775"
  dirmask     = "0775"
  ntlmv1_auth = false
}
```

## Argument Reference

The following arguments are supported:

* `netbiosname` - (Optional) NetBIOS name of the server. Default: `truenas`.
* `workgroup` - (Optional) Windows workgroup name. Default: `WORKGROUP`.
* `description` - (Optional) Server description. Default: `TrueNAS Server`.
* `enable_smb1` - (Optional) Enable SMB1 protocol support. Default: `false`.
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
