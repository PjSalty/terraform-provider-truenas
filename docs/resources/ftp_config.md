---
page_title: "truenas_ftp_config Resource - terraform-provider-truenas"
subcategory: "Sharing"
description: |-
  Manages the FTP service configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.
---

# truenas_ftp_config (Resource)

Manages the FTP service configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_ftp_config" "this" {
  port          = 21
  clients       = 32
  ipconnections = 5
  loginattempt  = 3
  timeout       = 600
  rootlogin     = false
  onlyanonymous = false
  onlylocal     = false
  defaultroot   = true
  ident         = false
  fxp           = false
  resume        = false
  ssltls_policy = "on"
}
```

## Argument Reference

The following arguments are supported:

* `port` - (Optional) FTP port. Default: `21`.
* `clients` - (Optional) Maximum number of simultaneous clients. Default: `5`.
* `ipconnections` - (Optional) Maximum connections per IP address (0 = unlimited). Default: `2`.
* `loginattempt` - (Optional) Maximum login attempts before disconnect. Default: `1`.
* `timeout` - (Optional) Timeout in seconds for idle connections. Default: `600`.
* `onlyanonymous` - (Optional) Allow only anonymous logins. Default: `false`.
* `onlylocal` - (Optional) Allow only local user logins. Default: `false`.
* `banner` - (Optional) FTP banner message. Default: ``.
* `filemask` - (Optional) File creation mask (umask). Default: `077`.
* `dirmask` - (Optional) Directory creation mask (umask). Default: `022`.
* `fxp` - (Optional) Enable FXP (File eXchange Protocol). Default: `false`.
* `resume` - (Optional) Allow transfer resume. Default: `false`.
* `defaultroot` - (Optional) Chroot users to their home directory. Default: `true`.
* `tls` - (Optional) Enable TLS for FTP connections. Default: `false`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_ftp_config` resource.

## Import

The `truenas_ftp_config` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_ftp_config.this singleton
```
