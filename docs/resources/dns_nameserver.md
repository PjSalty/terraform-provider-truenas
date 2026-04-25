---
page_title: "truenas_dns_nameserver Resource - terraform-provider-truenas"
subcategory: "Network"
description: |-
  Manages DNS nameserver configuration on TrueNAS SCALE. This is a singleton resource (only one instance should exist).
---

# truenas_dns_nameserver (Resource)

Manages DNS nameserver configuration on TrueNAS SCALE. This is a singleton resource (only one instance should exist).

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_dns_nameserver" "primary" {
  nameserver1 = "1.1.1.1"
  nameserver2 = "9.9.9.9"
  nameserver3 = "8.8.8.8"
}
```

## Argument Reference

The following arguments are supported:

* `nameserver1` - (Optional) Primary DNS nameserver IP address.
* `nameserver2` - (Optional) Secondary DNS nameserver IP address.
* `nameserver3` - (Optional) Tertiary DNS nameserver IP address.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_dns_nameserver` resource.

## Import

The `truenas_dns_nameserver` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_dns_nameserver.primary singleton
```
