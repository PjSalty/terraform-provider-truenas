---
page_title: "truenas_network_interface Resource - terraform-provider-truenas"
subcategory: "Network"
description: |-
  Manages a virtual network interface (BRIDGE, LINK_AGGREGATION, or VLAN) on TrueNAS SCALE. Changes go through a staged commit+checkin workflow which this resource handles automatically.
---

# truenas_network_interface (Resource)

Manages a virtual network interface (BRIDGE, LINK_AGGREGATION, or VLAN) on TrueNAS SCALE. Changes go through a staged commit+checkin workflow which this resource handles automatically.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_network_interface" "example" {
  type      = "PHYSICAL"
  name      = "eno1"
  dhcp      = false
  ipv6_auto = false

  aliases = [
    {
      type    = "INET"
      address = "10.0.0.20"
      netmask = 24
    }
  ]
  mtu = 1500
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The interface type: BRIDGE, LINK_AGGREGATION, or VLAN. Valid values: `PHYSICAL`, `BRIDGE`, `LINK_AGGREGATION`, `VLAN`. Changing this attribute forces a new resource to be created.
* `name` - (Optional) The interface name. If not provided, TrueNAS auto-generates one based on type (e.g. br0, bond1, vlan0). Changing this attribute forces a new resource to be created.
* `description` - (Optional) Human-readable description of the interface.
* `ipv4_dhcp` - (Optional) Enable IPv4 DHCP for automatic IP address assignment.
* `ipv6_auto` - (Optional) Enable IPv6 autoconfiguration.
* `mtu` - (Optional) Maximum transmission unit (68-9216 bytes).
* `aliases` - (Optional) List of IP address aliases to configure on the interface.
* `bridge_members` - (Optional) List of interfaces to add as members of this bridge. Only valid for type=BRIDGE.
* `lag_protocol` - (Optional) Link aggregation protocol: LACP, FAILOVER, LOADBALANCE, ROUNDROBIN, NONE. Only valid for type=LINK_AGGREGATION. Valid values: `LACP`, `FAILOVER`, `LOADBALANCE`, `ROUNDROBIN`, `NONE`.
* `lag_ports` - (Optional) List of interface names in the link aggregation group. Only valid for type=LINK_AGGREGATION.
* `vlan_parent_interface` - (Optional) Parent interface name for VLAN configuration. Only valid for type=VLAN.
* `vlan_tag` - (Optional) VLAN tag number (1-4094). Only valid for type=VLAN.
* `vlan_pcp` - (Optional) Priority Code Point for VLAN traffic (0-7). Only valid for type=VLAN.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_network_interface` resource.

## Import

The `truenas_network_interface` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_network_interface.example eno1
```
