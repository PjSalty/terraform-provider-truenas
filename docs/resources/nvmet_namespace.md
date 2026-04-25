---
page_title: "truenas_nvmet_namespace Resource - terraform-provider-truenas"
subcategory: "NVMe-oF"
description: |-
  Manages an NVMe-oF namespace within a subsystem on TrueNAS SCALE. A namespace exposes a ZVOL or file as a block device to connected hosts.
---

# truenas_nvmet_namespace (Resource)

Manages an NVMe-oF namespace within a subsystem on TrueNAS SCALE. A namespace exposes a ZVOL or file as a block device to connected hosts.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_zvol" "nvme_ns" {
  pool    = "tank"
  name    = "vols/nvme-ns0"
  volsize = 10737418240
  sparse  = true
}

resource "truenas_nvmet_subsys" "ns" {
  name = "nsexample"
}

resource "truenas_nvmet_namespace" "example" {
  subsys_id   = tonumber(truenas_nvmet_subsys.ns.id)
  device_type = "ZVOL"
  device_path = "zvol/tank/vols/nvme-ns0"
  enabled     = true
}
```

## Argument Reference

The following arguments are supported:

* `subsys_id` - (Required) ID of the NVMe-oF subsystem to contain this namespace.
* `device_type` - (Required) Type of device backing the namespace: ZVOL or FILE. Valid values: `ZVOL`, `FILE`.
* `device_path` - (Required) Path to the device or file for the namespace.
* `nsid` - (Optional) Namespace ID (NSID), unique within the subsystem. Auto-assigned if not provided.
* `filesize` - (Optional) Size of the backing file in bytes. Only used when device_type is FILE.
* `enabled` - (Optional) If false, the namespace is not accessible.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_nvmet_namespace` resource.

## Import

The `truenas_nvmet_namespace` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_nvmet_namespace.example 1
```
