---
page_title: "truenas_kmip_config Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages the KMIP (Key Management Interoperability Protocol) singleton configuration on TrueNAS SCALE.
---

# truenas_kmip_config (Resource)

Manages the KMIP (Key Management Interoperability Protocol) singleton configuration on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_kmip_config" "this" {
  server                = "kmip.example.com"
  port                  = 5696
  certificate           = 1 # Certificate resource ID
  certificate_authority = 2 # CA resource ID
  manage_sed_disks      = false
  manage_zfs_keys       = false
  enabled               = false
}
```

## Argument Reference

The following arguments are supported:

* `enabled` - (Optional) Whether KMIP functionality is enabled. Default: `false`.
* `manage_sed_disks` - (Optional) Use KMIP to manage Self-Encrypting Drive keys. Default: `false`.
* `manage_zfs_keys` - (Optional) Use KMIP to manage ZFS encryption keys. Default: `false`.
* `certificate` - (Optional) ID of the client certificate used for KMIP authentication (0 = none). Default: `0`.
* `certificate_authority` - (Optional) ID of the CA used to verify the KMIP server (0 = none). Default: `0`.
* `port` - (Optional) TCP port for the KMIP server connection. Default: `5696`.
* `server` - (Optional) Hostname or IP of the KMIP server. Empty string disables. Default: ``.
* `ssl_version` - (Optional) SSL/TLS protocol version. One of PROTOCOL_TLSv1, PROTOCOL_TLSv1_1, PROTOCOL_TLSv1_2. Valid values: `PROTOCOL_TLSv1`, `PROTOCOL_TLSv1_1`, `PROTOCOL_TLSv1_2`. Default: `PROTOCOL_TLSv1_2`.
* `change_server` - (Optional) Flag indicating the KMIP server endpoint is being changed. Default: `false`.
* `validate` - (Optional) Validate the KMIP server connection before saving. Default: `true`.
* `force_clear` - (Optional) Force clear existing keys when disabling KMIP. Default: `false`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_kmip_config` resource.

## Import

The `truenas_kmip_config` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_kmip_config.this singleton
```
