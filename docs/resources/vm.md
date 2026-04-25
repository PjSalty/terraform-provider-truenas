---
page_title: "truenas_vm Resource - terraform-provider-truenas"
subcategory: "Virtualization"
description: |-
  Manages a TrueNAS SCALE virtual machine via the /vm API. Devices (disks, NICs, CDROMs, displays) are managed independently via truenas_vm_device. Default timeouts: 20m for create/update/delete (VM start/stop can be slow).
---

# truenas_vm (Resource)

Manages a TrueNAS SCALE virtual machine via the /vm API. Devices (disks, NICs, CDROMs, displays) are managed independently via truenas_vm_device. Default timeouts: 20m for create/update/delete (VM start/stop can be slow).

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_vm" "example" {
  name             = "examplevm"
  description      = "Example VM managed by Terraform"
  vcpus            = 2
  cores            = 2
  threads          = 1
  memory           = 4096 # MiB
  bootloader       = "UEFI"
  autostart        = true
  time             = "LOCAL"
  shutdown_timeout = 90
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the VM. Only alphanumeric characters are allowed — no spaces, hyphens, underscores, or punctuation.
* `memory` - (Required) Memory allocated to the VM, in MiB. Minimum 20 MiB. Max 4 TiB.
* `description` - (Optional) A description for the VM. Default: ``.
* `vcpus` - (Optional) Number of virtual CPU sockets (1-16). Default: `1`.
* `cores` - (Optional) Number of cores per CPU socket (1-254). Default: `1`.
* `threads` - (Optional) Number of threads per core (1-254). Default: `1`.
* `min_memory` - (Optional) Optional minimum memory (MiB) for memory ballooning.
* `bootloader` - (Optional) Bootloader to use: UEFI or UEFI_CSM. Valid values: `UEFI`, `UEFI_CSM`. Default: `UEFI`.
* `bootloader_ovmf` - (Optional) OVMF firmware file name. Default: `OVMF_CODE.fd`.
* `autostart` - (Optional) Whether the VM should start automatically on host boot. Default: `true`.
* `hide_from_msr` - (Optional) Hide the KVM hypervisor from MSR-based discovery (for GPU passthrough). Default: `false`.
* `ensure_display_device` - (Optional) Ensure the VM always has a display device attached. Default: `true`.
* `time` - (Optional) VM clock source: LOCAL or UTC. Valid values: `LOCAL`, `UTC`. Default: `LOCAL`.
* `shutdown_timeout` - (Optional) Seconds to wait for the VM to cleanly shut down before forcing power off (5-300). Default: `90`.
* `arch_type` - (Optional) Guest architecture (nullable; system chooses default when unset).
* `machine_type` - (Optional) Guest machine type (nullable; system chooses default when unset).
* `uuid` - (Optional) VM UUID. If unset, TrueNAS generates one.
* `command_line_args` - (Optional) Additional QEMU command line arguments. Default: ``.
* `cpu_mode` - (Optional) CPU mode: CUSTOM, HOST-MODEL, or HOST-PASSTHROUGH. Valid values: `CUSTOM`, `HOST-MODEL`, `HOST-PASSTHROUGH`. Default: `CUSTOM`.
* `cpu_model` - (Optional) CPU model when cpu_mode is CUSTOM (nullable).
* `cpuset` - (Optional) Host CPU set to pin the VM to (nullable).
* `nodeset` - (Optional) NUMA node set to pin the VM to (nullable).
* `pin_vcpus` - (Optional) Pin vCPUs to host CPUs listed in cpuset. Default: `false`.
* `suspend_on_snapshot` - (Optional) Suspend the VM automatically while periodic snapshots run. Default: `false`.
* `trusted_platform_module` - (Optional) Attach a virtual TPM to the VM. Default: `false`.
* `hyperv_enlightenments` - (Optional) Enable Hyper-V enlightenments for Windows guests. Default: `false`.
* `enable_secure_boot` - (Optional) Enable UEFI Secure Boot. Default: `false`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_vm` resource.
* `status` - Current VM state (RUNNING, STOPPED, etc.). Read-only.

## Import

The `truenas_vm` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_vm.example 1
```
