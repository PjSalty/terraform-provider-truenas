---
page_title: "truenas_acme_dns_authenticator Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages an ACME DNS authenticator on TrueNAS SCALE for Let's Encrypt certificate validation.
---

# truenas_acme_dns_authenticator (Resource)

Manages an ACME DNS authenticator on TrueNAS SCALE for Let's Encrypt certificate validation.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_acme_dns_authenticator" "example" {
  name          = "cloudflare"
  authenticator = "cloudflare"
  attributes = {
    cloudflare_email   = "dns@example.com"
    cloudflare_api_key = "changeme"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ACME DNS authenticator.
* `authenticator` - (Required) The DNS provider type (e.g., cloudflare, route53, digitalocean). Changing this attribute forces a new resource to be created.
* `attributes` - (Optional) Provider-specific configuration attributes (e.g., api_token for Cloudflare). Marked sensitive.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_acme_dns_authenticator` resource.

## Import

The `truenas_acme_dns_authenticator` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_acme_dns_authenticator.example 1
```
