---
page_title: "truenas_network_interface Resource - terraform-provider-truenas"
subcategory: "Network"
description: |-
  Manages a network interface on TrueNAS SCALE. Virtual types (BRIDGE, LINK_AGGREGATION, VLAN) support full create/update/delete. For type=PHYSICAL, create adopts the existing NIC and configures it in place, and destroy removes the resource from state but leaves the live configuration untouched. Changes go through a staged commit+checkin workflow which this resource handles automatically.
---

# truenas_network_interface (Resource)

Manages a network interface on TrueNAS SCALE. Virtual types (BRIDGE, LINK_AGGREGATION, VLAN) support full create/update/delete. For type=PHYSICAL, create adopts the existing NIC and configures it in place, and destroy removes the resource from state but leaves the live configuration untouched. Changes go through a staged commit+checkin workflow which this resource handles automatically.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## PHYSICAL interfaces

Hardware NICs are discovered by the host and cannot be created or deleted
through the TrueNAS API, so this resource treats `type = "PHYSICAL"`
differently from the virtual types:

* **Create** adopts the existing NIC named by `name` and applies the plan
  settings (aliases, `ipv4_dhcp`, `ipv6_auto`, `description`, `mtu`) via
  `interface.update`. If no NIC with that name exists, create fails with a
  not-found error. Virtual-only arguments (`bridge_members`, `lag_*`,
  `vlan_*`) are never sent for a physical interface.
* **Destroy** removes the resource from Terraform state only. The live
  interface keeps its configuration because the hardware cannot be deleted.
* **Import** is supported by interface name, for example
  `terraform import truenas_network_interface.example eno1`.

## Example Usage

### Basic

```terraform
resource "truenas_network_interface" "example" {
  type      = "PHYSICAL"
  name      = "eno1"
  ipv4_dhcp = false
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

* `type` - (Required) The interface type: PHYSICAL, BRIDGE, LINK_AGGREGATION, or VLAN. PHYSICAL adopts an existing hardware NIC (identified by name) instead of creating one. Valid values: `PHYSICAL`, `BRIDGE`, `LINK_AGGREGATION`, `VLAN`. Changing this attribute forces a new resource to be created.
* `name` - (Optional) The interface name. Required for type=PHYSICAL (names the existing NIC to adopt). If not provided for virtual types, TrueNAS auto-generates one based on type (e.g. br0, bond1, vlan0). Changing this attribute forces a new resource to be created.
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
* `rollback` - (Optional) Commit changes with a rollback safety window (defaults to `true`). TrueNAS reverts the change automatically unless it is checked in within 60 seconds; the provider sends the checkin immediately after a successful commit. When `false`, changes are applied immediately without a rollback timer. Disabling rollback is faster but riskier: a bad change that breaks networking may make the interface permanently unreachable via the management network.
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
