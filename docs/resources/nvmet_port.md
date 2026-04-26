---
page_title: "truenas_nvmet_port Resource - terraform-provider-truenas"
subcategory: "NVMe-oF"
description: |-
  Manages an NVMe-oF transport port on TrueNAS SCALE.
---

# truenas_nvmet_port (Resource)

Manages an NVMe-oF transport port on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_nvmet_port" "example" {
  addr_trtype  = "TCP"
  addr_traddr  = "10.0.0.20"
  addr_trsvcid = 4420
  addr_adrfam  = "IPV4"
}
```

## Argument Reference

The following arguments are supported:

* `addr_trtype` - (Required) Fabric transport technology: TCP, RDMA, or FC. Valid values: `TCP`, `RDMA`, `FC`.
* `addr_traddr` - (Required) Transport address. For TCP/RDMA, an IPv4/IPv6 address. For FC, a fabric-specific address.
* `addr_trsvcid` - (Optional) Transport service ID. For TCP/RDMA, the port number (default 4420). Not used for FC.
* `inline_data_size` - (Optional) Maximum size for inline data transfers.
* `max_queue_size` - (Optional) Maximum number of queue entries.
* `pi_enable` - (Optional) Whether Protection Information (PI) is enabled.
* `enabled` - (Optional) Whether the port is enabled.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_nvmet_port` resource.
* `index` - Internal port index.

## Import

The `truenas_nvmet_port` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_nvmet_port.example 1
```
