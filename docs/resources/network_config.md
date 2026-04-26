---
page_title: "truenas_network_config Resource - terraform-provider-truenas"
subcategory: "Network"
description: |-
  Manages the global network configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.
---

# truenas_network_config (Resource)

Manages the global network configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_network_config" "this" {
  hostname        = "truenas"
  domain          = "example.com"
  nameserver1     = "1.1.1.1"
  nameserver2     = "9.9.9.9"
  ipv4gateway     = "10.0.0.1"
  httpproxy       = ""
  netwait_enabled = false
}
```

## Argument Reference

The following arguments are supported:

* `hostname` - (Optional) The system hostname.
* `domain` - (Optional) The system domain. Default: `local`.
* `ipv4gateway` - (Optional) IPv4 default gateway. Default: ``.
* `ipv6gateway` - (Optional) IPv6 default gateway. Default: ``.
* `nameserver1` - (Optional) Primary DNS nameserver. Default: ``.
* `nameserver2` - (Optional) Secondary DNS nameserver. Default: ``.
* `nameserver3` - (Optional) Tertiary DNS nameserver. Default: ``.
* `httpproxy` - (Optional) HTTP proxy URL. Default: ``.
* `hosts` - (Optional) Additional hosts entries.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_network_config` resource.

## Import

The `truenas_network_config` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_network_config.this singleton
```
