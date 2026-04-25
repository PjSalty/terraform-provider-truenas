---
page_title: "truenas_nvmet_port_subsys Resource - terraform-provider-truenas"
subcategory: "NVMe-oF"
description: |-
  Manages an NVMe-oF port-to-subsystem association. This makes a subsystem accessible via a given transport port. Both port_id and subsys_id require replacement if changed.
---

# truenas_nvmet_port_subsys (Resource)

Manages an NVMe-oF port-to-subsystem association. This makes a subsystem accessible via a given transport port. Both port_id and subsys_id require replacement if changed.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_nvmet_port" "p" {
  addr_trtype  = "TCP"
  addr_traddr  = "10.0.0.20"
  addr_trsvcid = 4420
}

resource "truenas_nvmet_subsys" "s2" {
  name = "portsubsysexample"
}

resource "truenas_nvmet_port_subsys" "example" {
  port_id   = tonumber(truenas_nvmet_port.p.id)
  subsys_id = tonumber(truenas_nvmet_subsys.s2.id)
}
```

## Argument Reference

The following arguments are supported:

* `port_id` - (Required) ID of the NVMe-oF port to associate. Changing this attribute forces a new resource to be created.
* `subsys_id` - (Required) ID of the NVMe-oF subsystem to expose on the port. Changing this attribute forces a new resource to be created.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_nvmet_port_subsys` resource.

## Import

The `truenas_nvmet_port_subsys` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_nvmet_port_subsys.example 1
```
