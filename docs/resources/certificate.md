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

### Import a certificate you already hold

```terraform
resource "truenas_certificate" "example" {
  name        = "wildcard-example-com"
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = file("${path.module}/cert.pem")
  privatekey  = file("${path.module}/key.pem")
}
```

### Generate a signing request

`san` is required and must list every name the certificate is for. A common
name on its own is refused. An RSA key also needs `key_length`.

```terraform
resource "truenas_certificate" "csr" {
  name             = "app-example-com-csr"
  create_type      = "CERTIFICATE_CREATE_CSR"
  common           = "app.example.com"
  san              = ["app.example.com", "www.app.example.com"]
  key_type         = "RSA"
  key_length       = 4096
  digest_algorithm = "SHA256"
  country          = "US"
  state            = "California"
  city             = "San Jose"
  organization     = "Example"
  email            = "hostmaster@example.com"
}
```

Write `san` entries as bare names. TrueNAS reads them back off the parsed
certificate with the general-name kind attached (`DNS:app.example.com`), and
the provider treats the two spellings as the same value rather than as a
permanent diff.

### Satisfy a signing request through ACME

ACME needs an existing CSR and an authenticator for every domain in it.

```terraform
resource "truenas_acme_dns_authenticator" "cloudflare" {
  name          = "cloudflare"
  authenticator = "cloudflare"
  attributes = {
    cloudflare_email   = "dns@example.com"
    cloudflare_api_key = var.cloudflare_api_key
  }
}

resource "truenas_certificate" "acme" {
  name               = "app-example-com"
  create_type        = "CERTIFICATE_CREATE_ACME"
  csr_id             = truenas_certificate.csr.id
  acme_directory_uri = "https://acme-v02.api.letsencrypt.org/directory"
  tos                = true
  renew_days         = 10

  dns_mapping = {
    "app.example.com"     = truenas_acme_dns_authenticator.cloudflare.id
    "www.app.example.com" = truenas_acme_dns_authenticator.cloudflare.id
  }
}
```

## Arguments by create_type

TrueNAS validates the request against a different model per `create_type`, so
an argument that is required for one is rejected on another. The provider
checks both directions at plan time.

| `create_type` | Required | Rejected |
|---|---|---|
| `CERTIFICATE_CREATE_IMPORTED` | `certificate`, `privatekey` | the ACME arguments |
| `CERTIFICATE_CREATE_CSR` | `san` (at least one), plus `key_length` unless `key_type = "EC"` | the ACME arguments |
| `CERTIFICATE_CREATE_IMPORTED_CSR` | `certificate` | the ACME arguments |
| `CERTIFICATE_CREATE_ACME` | `tos`, `csr_id`, `acme_directory_uri`, `dns_mapping` | - |

`renew_days` is the one ACME argument that is optional; TrueNAS defaults it to
10. `key_type` and `digest_algorithm` are optional for a CSR for the same
reason, defaulting to `RSA` and `SHA256`.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the certificate.
* `create_type` - (Required) The certificate creation type: CERTIFICATE_CREATE_IMPORTED, CERTIFICATE_CREATE_CSR, CERTIFICATE_CREATE_IMPORTED_CSR, or CERTIFICATE_CREATE_ACME. Valid values: `CERTIFICATE_CREATE_IMPORTED`, `CERTIFICATE_CREATE_CSR`, `CERTIFICATE_CREATE_IMPORTED_CSR`, `CERTIFICATE_CREATE_ACME`. Changing this attribute forces a new resource to be created.
* `certificate` - (Optional) The PEM-encoded certificate data. Required for CERTIFICATE_CREATE_IMPORTED. Changing this attribute forces a new resource to be created.
* `privatekey` - (Optional) The PEM-encoded private key. Required for CERTIFICATE_CREATE_IMPORTED. Changing this attribute forces a new resource to be created. Marked sensitive.
* `key_type` - (Optional) The key type: RSA or EC. Valid values: `RSA`, `EC`. Changing this attribute forces a new resource to be created.
* `key_length` - (Optional) The key length in bits. Required for CERTIFICATE_CREATE_CSR unless `key_type` is `EC`. Valid values: `2048`, `4096`.
* `digest_algorithm` - (Optional) The digest algorithm (e.g., SHA256, SHA384). Valid values: `SHA224`, `SHA256`, `SHA384`, `SHA512`.
* `lifetime` - (Optional) The certificate lifetime in days (1-36500).
* `country` - (Optional) The certificate country (C). Two-letter ISO 3166 code.
* `state` - (Optional) The certificate state/province (ST).
* `city` - (Optional) The certificate city/locality (L).
* `organization` - (Optional) The certificate organization (O).
* `organizational_unit` - (Optional) The certificate organizational unit (OU).
* `email` - (Optional) The certificate email address.
* `common` - (Optional) The common name (CN) of the certificate.
* `san` - (Optional) Subject alternative names, written as bare values such as `example.com`. Required for CERTIFICATE_CREATE_CSR, which needs at least one entry.
* `tos` - (Optional) CERTIFICATE_CREATE_ACME only, and required for it. Accept the ACME provider's terms of service.
* `csr_id` - (Optional) CERTIFICATE_CREATE_ACME only, and required for it. ID of an existing certificate signing request to satisfy, typically another `truenas_certificate` created with CERTIFICATE_CREATE_CSR.
* `acme_directory_uri` - (Optional) CERTIFICATE_CREATE_ACME only, and required for it. The ACME directory URI, for example `https://acme-v02.api.letsencrypt.org/directory`.
* `renew_days` - (Optional) CERTIFICATE_CREATE_ACME only. Days before expiry to attempt renewal, between 1 and 30. Defaults to 10 when unset.
* `dns_mapping` - (Optional) CERTIFICATE_CREATE_ACME only, and required for it. Maps each domain in the CSR to the ID of a `truenas_acme_dns_authenticator` that can complete the DNS-01 challenge for it. The keys are the CSR's `common` and `san` values as written there, and every one of them must appear: TrueNAS refuses the request naming any domain it cannot authenticate, and refuses a key that is not in the CSR.
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
