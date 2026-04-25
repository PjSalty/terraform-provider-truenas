---
page_title: "truenas_snmp_config Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages the SNMP configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.
---

# truenas_snmp_config (Resource)

Manages the SNMP configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_snmp_config" "this" {
  location  = "Rack 1"
  contact   = "ops@example.com"
  community = "public"
  v3        = false
  traps     = false
  zilstat   = false
  loglevel  = 3
}
```

## Argument Reference

The following arguments are supported:

* `community` - (Optional) SNMP community string. Default: `public`.
* `contact` - (Optional) SNMP contact information. Default: ``.
* `location` - (Optional) SNMP system location. Default: ``.
* `v3` - (Optional) Enable SNMPv3 support. Default: `false`.
* `v3_username` - (Optional) SNMPv3 username. Default: ``.
* `v3_authtype` - (Optional) SNMPv3 authentication type (SHA, MD5). Valid values: `SHA`, `MD5`. Default: `SHA`.
* `v3_password` - (Optional) SNMPv3 authentication password. Default: ``. Marked sensitive.
* `v3_privproto` - (Optional) SNMPv3 privacy protocol (AES, DES). Valid values: `AES`, `DES`.
* `v3_privpassphrase` - (Optional) SNMPv3 privacy passphrase. Marked sensitive.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_snmp_config` resource.

## Import

The `truenas_snmp_config` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_snmp_config.this singleton
```
