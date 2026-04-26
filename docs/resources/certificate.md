---
page_title: "truenas_certificate Resource - terraform-provider-truenas"
subcategory: "Users & RBAC"
description: |-
  Manages a TLS certificate on TrueNAS SCALE. Default timeouts: 20m create (ACME/CSR signing can be slow), 10m update/delete.
---

# truenas_certificate (Resource)

Manages a TLS certificate on TrueNAS SCALE. Default timeouts: 20m create (ACME/CSR signing can be slow), 10m update/delete.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_certificate" "example" {
  name        = "wildcard-example-com"
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = file("${path.module}/cert.pem")
  privatekey  = file("${path.module}/key.pem")
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the certificate.
* `create_type` - (Required) The certificate creation type: CERTIFICATE_CREATE_IMPORTED, CERTIFICATE_CREATE_CSR, CERTIFICATE_CREATE_IMPORTED_CSR, or CERTIFICATE_CREATE_ACME. Valid values: `CERTIFICATE_CREATE_IMPORTED`, `CERTIFICATE_CREATE_CSR`, `CERTIFICATE_CREATE_IMPORTED_CSR`, `CERTIFICATE_CREATE_ACME`. Changing this attribute forces a new resource to be created.
* `certificate` - (Optional) The PEM-encoded certificate data. Required for CERTIFICATE_CREATE_IMPORTED. Changing this attribute forces a new resource to be created.
* `privatekey` - (Optional) The PEM-encoded private key. Required for CERTIFICATE_CREATE_IMPORTED. Changing this attribute forces a new resource to be created. Marked sensitive.
* `key_type` - (Optional) The key type: RSA or EC. Valid values: `RSA`, `EC`. Changing this attribute forces a new resource to be created.
* `key_length` - (Optional) The key length in bits (1024, 2048, 4096).
* `digest_algorithm` - (Optional) The digest algorithm (e.g., SHA256, SHA384). Valid values: `SHA224`, `SHA256`, `SHA384`, `SHA512`.
* `lifetime` - (Optional) The certificate lifetime in days (1-36500).
* `country` - (Optional) The certificate country (C). Two-letter ISO 3166 code.
* `state` - (Optional) The certificate state/province (ST).
* `city` - (Optional) The certificate city/locality (L).
* `organization` - (Optional) The certificate organization (O).
* `organizational_unit` - (Optional) The certificate organizational unit (OU).
* `email` - (Optional) The certificate email address.
* `common` - (Optional) The common name (CN) of the certificate.
* `san` - (Optional) Subject alternative names.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_certificate` resource.
* `dn` - The full distinguished name.
* `from` - The certificate valid-from date.
* `until` - The certificate valid-until date.
* `expired` - Whether the certificate has expired.

## Import

The `truenas_certificate` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_certificate.example 1
```
