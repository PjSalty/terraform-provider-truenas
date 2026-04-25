---
page_title: "truenas_kerberos_realm Resource - terraform-provider-truenas"
subcategory: "Auth & Integration"
description: |-
  Manages a Kerberos realm on TrueNAS. Realm names are case-sensitive and conventionally upper-case.
---

# truenas_kerberos_realm (Resource)

Manages a Kerberos realm on TrueNAS. Realm names are case-sensitive and conventionally upper-case.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_kerberos_realm" "example" {
  realm          = "EXAMPLE.COM"
  kdc            = ["kdc1.example.com", "kdc2.example.com"]
  admin_server   = ["kdc1.example.com"]
  kpasswd_server = ["kdc1.example.com"]
}
```

## Argument Reference

The following arguments are supported:

* `realm` - (Required) Kerberos realm name (e.g. EXAMPLE.COM).
* `primary_kdc` - (Optional) Optional primary/master KDC hostname. Used as a fallback when the machine password is invalid.
* `kdc` - (Optional) List of Kerberos KDC hostnames. If empty, libraries use DNS SRV lookups.
* `admin_server` - (Optional) List of Kerberos admin server hostnames.
* `kpasswd_server` - (Optional) List of Kerberos kpasswd server hostnames.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_kerberos_realm` resource.

## Import

The `truenas_kerberos_realm` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_kerberos_realm.example 1
```
