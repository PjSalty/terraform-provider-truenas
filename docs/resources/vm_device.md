---
page_title: "truenas_vm_device Resource - terraform-provider-truenas"
subcategory: "Virtualization"
description: |-
  Manages a device attached to a TrueNAS SCALE virtual machine. The dtype field selects the device type (DISK, NIC, CDROM, DISPLAY, RAW, PCI, USB) and the attributes map carries the type-specific fields.
---

# truenas_vm_device (Resource)

Manages a device attached to a TrueNAS SCALE virtual machine. The dtype field selects the device type (DISK, NIC, CDROM, DISPLAY, RAW, PCI, USB) and the attributes map carries the type-specific fields.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_vm" "example" {
  name   = "examplevm"
  memory = 4096
}

# Disk device backed by a zvol
resource "truenas_zvol" "example_disk" {
  pool    = "tank"
  name    = "vols/examplevm-disk0"
  volsize = 21474836480 # 20 GiB
  sparse  = true
}

resource "truenas_vm_device" "disk" {
  vm    = tonumber(truenas_vm.example.id)
  dtype = "DISK"
  attributes = {
    path = "/dev/zvol/tank/vols/examplevm-disk0"
    type = "VIRTIO"
  }
}

# NIC attached to the bridge
resource "truenas_vm_device" "nic" {
  vm    = tonumber(truenas_vm.example.id)
  dtype = "NIC"
  attributes = {
    type       = "VIRTIO"
    nic_attach = "br0"
  }
}
```

## Argument Reference

The following arguments are supported:

* `vm` - (Required) The ID of the VM this device is attached to.
* `dtype` - (Required) The device type: DISK, NIC, CDROM, DISPLAY, RAW, PCI, or USB. Changing this forces replacement. Valid values: `DISK`, `NIC`, `CDROM`, `DISPLAY`, `RAW`, `PCI`, `USB`. Changing this attribute forces a new resource to be created.
* `attributes` - (Required) Type-specific device attributes as string values. For example DISK uses path, type (AHCI/VIRTIO), iotype; NIC uses type, mac, nic_attach; DISPLAY uses resolution, bind, port, password, web; CDROM uses path; RAW uses path, type, size; PCI uses pptdev; USB uses controller_type, usb.
* `order` - (Optional) Device order on the VM's bus. Optional.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_vm_device` resource.

## Import

The `truenas_vm_device` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_vm_device.disk 1
```
