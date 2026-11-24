---
page_title: "truenas_resilver_config Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages the pool resilver priority window on TrueNAS SCALE. Singleton resource.
---

# truenas_resilver_config (Resource)

Manages the pool resilver priority window: the hours and days during which
TrueNAS lets a resilver use more of the disks, so that a replaced disk is back
in the pool sooner. Outside the window the ZFS defaults apply and client I/O
comes first.

This resource is a **singleton**, TrueNAS has exactly one window and it covers
every pool. In the TrueNAS UI it is **Storage > Resilver Priority**.

## Example Usage

```hcl
resource "truenas_resilver_config" "this" {
  enabled = true

  # 18:00 to 09:00: a begin later than the end rolls over midnight.
  begin = "18:00"
  end   = "09:00"

  # 1 is Monday, 7 is Sunday.
  weekday = [1, 2, 3, 4, 5, 6, 7]
}
```

## Argument Reference

Every argument is optional. Anything the configuration does not set is left
untouched on the server, matching the update semantics of the underlying API,
and is read back into state.

* `enabled` - (Optional) Whether the priority window is in force. When `false`
  TrueNAS stops changing the resilver tunables and leaves them as they are, see
  [Switching the window off](#switching-the-window-off).
* `begin` - (Optional) Start of the window as a zero-padded 24 hour time,
  `HH:MM`, in the system time zone. TrueNAS defaults to `18:00`.
* `end` - (Optional) End of the window, in the same form. TrueNAS defaults to
  `09:00`. A `begin` later than `end` rolls over midnight.
* `weekday` - (Optional) Set of days the window begins on, `1` (Monday)
  through `7` (Sunday). At least one day is required. TrueNAS defaults to every
  day. See [Windows that cross midnight](#windows-that-cross-midnight).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

`begin` and `end` are checked at plan time and must be zero-padded: write
`09:00`, not `9:00`. TrueNAS accepts the short form but reads every time back
as `HH:MM`, so the short form would show a change on every plan.

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

* `id` - Fixed singleton identifier. Always `"resilver_config"`.

## Import

```shell
terraform import truenas_resilver_config.this resilver_config
```

The ID is the literal string `resilver_config`; any other value is rejected.

## Behavior notes

### What the window changes

TrueNAS writes four ZFS module parameters when the window begins and ends, when
this configuration changes, and when the system starts:

| Parameter                   | Inside the window | Outside |
| --------------------------- | ----------------- | ------- |
| `zfs_resilver_min_time_ms`  | 3000              | 1500    |
| `zfs_vdev_scrub_max_active` | 8                 | 3       |
| `zfs_vdev_nia_credit`       | 10                | 5       |
| `zfs_vdev_nia_delay`        | 2                 | 5       |

The values are those of TrueNAS SCALE 25.10. They are system-wide, so every
pool gets them, and a scrub that runs inside the window is given the same
room as a resilver. Client I/O to a pool that is resilvering is slower inside
the window; that is the trade the window makes.

Do not also set these four parameters with `truenas_tunable` while the window
is enabled: TrueNAS overwrites them at each end of the window.

### Windows that cross midnight

A window whose `begin` is later than its `end` belongs to the day it begins
on, and runs into the next morning whether or not that next day is listed.
With `begin = "22:00"`, `end = "06:00"` and `weekday = [5]`, the window runs
from Friday 22:00 to Saturday 06:00.

### Switching the window off

`enabled = false` stops TrueNAS from changing the parameters; it does not put
the outside values back. Switched off while a window is open, the system keeps
the inside values until it restarts. To return to the outside values straight
away, switch it off outside the window, or first move the window so that the
current time falls outside it and switch it off in a second apply.

### Delete is a no-op

Removing this resource from configuration does not switch the window off. The
configuration is a singleton that always exists, so there is nothing to
delete; Terraform simply stops managing it. Set `enabled = false` and apply
before removing the resource if the window should go.
