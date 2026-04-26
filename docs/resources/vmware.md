---
page_title: "truenas_vmware Resource - terraform-provider-truenas"
subcategory: "Virtualization"
description: |-
  Manages a VMware host registration on TrueNAS SCALE for snapshot-aware replication.
---

# truenas_vmware (Resource)

Manages a VMware host registration on TrueNAS SCALE for snapshot-aware replication.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_vmware" "example" {
  hostname   = "vcenter.example.com"
  username   = "administrator@vsphere.local"
  password   = "ChangeMe!2026"
  datastore  = "truenas-nfs"
  filesystem = "tank/vmware"
}
```

## Argument Reference

The following arguments are supported:

* `datastore` - (Required) Datastore name that exists on the VMware host.
* `filesystem` - (Required) ZFS filesystem or dataset used for VMware storage.
* `hostname` - (Required) VMware host (or vCenter) hostname or IP address.
* `username` - (Required) Username used to authenticate to the VMware host.
* `password` - (Required) Password used to authenticate to the VMware host. Marked sensitive.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_vmware` resource.

## Import

The `truenas_vmware` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_vmware.example 1
```
