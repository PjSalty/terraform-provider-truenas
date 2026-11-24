---
page_title: "truenas_disk Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages the standby timer, power management level and description of a disk TrueNAS already knows about.
---

# truenas_disk (Resource)

Manages the stored settings of one disk TrueNAS already knows about: its
standby timer, its Advanced Power Management level and its description.

A disk is hardware, so this resource never creates or destroys one. Creating
it **adopts** the disk named by `identifier` and fails if there is no such
disk. Destroying it only stops Terraform managing the disk, see
[Delete leaves the disk alone](#delete-leaves-the-disk-alone).

## Example Usage

```hcl
# The device name (sda) can change across restarts; the identifier follows
# the drive. Look the identifier up from the name with the data source:
data "truenas_disk" "bay3" {
  name = "sda"
}

resource "truenas_disk" "bay3" {
  identifier = data.truenas_disk.bay3.identifier

  description = "bay 3, archive pool"

  # Spin down after 30 minutes without I/O. Left out, the current value is
  # kept.
  hddstandby = "30"

  # Advanced Power Management: 128 and above never spin the drive down.
  advpowermgmt = "128"
}
```

## Argument Reference

* `identifier` - (Required) The TrueNAS disk identifier, for example
  `{serial_lunid}WD-WCC4N1234567_5000cca2`. It is the `identifier` attribute of
  the `truenas_disk` data source, which finds a disk by its device name. It is
  also the import ID. Changing it forces a new resource.
* `hddstandby` - (Optional) Minutes of inactivity before the drive spins down:
  one of `ALWAYS ON` (never), `5`, `10`, `20`, `30`, `60`, `120`, `180`, `240`,
  `300` or `330`. TrueNAS defaults to `ALWAYS ON`.
* `advpowermgmt` - (Optional) Advanced Power Management level: one of
  `DISABLED`, `1`, `64`, `127`, `128`, `192` or `254`. `1` to `127` allow the
  drive to spin down, `128` and above do not, and a higher number favors
  performance. TrueNAS defaults to `DISABLED`.
* `description` - (Optional) Free-text description shown in the TrueNAS UI.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

Every setting is optional. One the configuration leaves out is not sent and
keeps the value the disk has, which is read back into state. Both
enumerations are checked at plan time and are case-sensitive: TrueNAS reads
them back upper-cased, so `Always On` would show a change on every plan.

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

* `id` - The disk identifier, the same value as `identifier`.
* `name` - The current device name, for example `sda`. It is not stable: Linux
  can name the same drive differently after a restart. A new name shows up on
  the next refresh and is not a change to configure.
* `serial` - The disk serial number.
* `model` - The disk model. Empty when TrueNAS does not report one.
* `size` - The disk size in bytes. `0` when TrueNAS does not report one.
* `type` - The disk type, `SSD` or `HDD`. Empty when TrueNAS has not determined one.
* `pool` - The name of the pool the disk is a member of, or null when it is in none.

## Import

```shell
terraform import truenas_disk.bay3 '{serial_lunid}WD-WCC4N1234567_5000cca2'
```

The ID is the disk identifier. Quote it in a shell: it starts with a brace.

## Behavior notes

### What TrueNAS does with a change

TrueNAS stores the settings and, only when the standby timer or the power
level changed, runs `hdparm -B` (power management) at once and `hdparm -S`
(standby) a minute later against the drive, in the background. Their exit
status is ignored, so a drive or controller that does not support them, such
as a virtual disk, does not make the apply fail. The values still read back as
set, which does not prove the drive honoured them.

TrueNAS also applies the settings to every disk when it starts, except on
TrueNAS Enterprise.

### Settings left out

`disk.update` also accepts the bus, number, LUN ID, pool, enclosure and the
password of a self-encrypting drive. This resource sends none of them: the
first five describe the hardware or are overwritten by TrueNAS when it syncs
disks, and the password is a secret that is write-only. Manage a SED password
in the TrueNAS UI.

### Delete leaves the disk alone

Removing this resource from configuration sends nothing to TrueNAS. The
standby timer, power level and description stay as they are, nothing is wiped,
and the disk stays in its pool. Set the values you want back and apply before
removing the resource.

### A disk that disappears

If a refresh finds no disk with the identifier, for example because the drive
was pulled and TrueNAS has expired it, the resource is dropped from state and
the next plan proposes adopting it again, which fails until the disk is back.
