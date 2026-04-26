---
page_title: "truenas_static_route Resource - terraform-provider-truenas"
subcategory: "Network"
description: |-
  Manages a static network route on TrueNAS SCALE.
---

# truenas_static_route (Resource)

Manages a static network route on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_static_route" "example" {
  destination = "10.99.0.0/16"
  gateway     = "10.0.0.1"
  description = "Route to backup LAN"
}
```

## Argument Reference

The following arguments are supported:

* `destination` - (Required) The destination network in CIDR notation (e.g., 192.168.1.0/24).
* `gateway` - (Required) The gateway IP address for the route.
* `description` - (Optional) A description for the static route. Default: ``.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_static_route` resource.

## Import

The `truenas_static_route` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_static_route.example 1
```
