---
page_title: "truenas_kerberos_keytab Resource - terraform-provider-truenas"
subcategory: "Auth & Integration"
description: |-
  Manages a Kerberos keytab entry on TrueNAS. Uploaded keytabs are merged into the system keytab at /etc/krb5.keytab.
---

# truenas_kerberos_keytab (Resource)

Manages a Kerberos keytab entry on TrueNAS. Uploaded keytabs are merged into the system keytab at /etc/krb5.keytab.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_kerberos_keytab" "example" {
  name = "truenas-host"
  # base64-encoded keytab file contents
  file = filebase64("${path.module}/truenas.keytab")
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Identifier for this keytab entry (e.g. SERVICE_PRINCIPAL). Note that names like AD_MACHINE_ACCOUNT and IPA_MACHINE_ACCOUNT are reserved.
* `file` - (Required) Base64-encoded keytab file contents. Marked sensitive.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_kerberos_keytab` resource.

## Import

The `truenas_kerberos_keytab` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_kerberos_keytab.example 1
```
