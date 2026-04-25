---
page_title: "truenas_rsync_task Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages an rsync task on TrueNAS SCALE.
---

# truenas_rsync_task (Resource)

Manages an rsync task on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_rsync_task" "example" {
  path         = "/mnt/tank/backup"
  user         = "root"
  mode         = "SSH"
  remotehost   = "backup.example.com"
  remotepath   = "/srv/backup"
  direction    = "PUSH"
  enabled      = true
  recursive    = true
  times        = true
  compress     = true
  archive      = true
  delete       = false
  preserveperm = true
  schedule {
    minute = "0"
    hour   = "2"
  }
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The local path to sync (e.g., /mnt/tank/data).
* `user` - (Required) The user to run the rsync task as.
* `remotehost` - (Optional) The remote host to sync with.
* `remoteport` - (Optional) The remote SSH port. Default: `22`.
* `mode` - (Optional) Rsync mode: SSH or MODULE. Valid values: `SSH`, `MODULE`. Default: `MODULE`.
* `remotemodule` - (Optional) The remote rsync module name (for MODULE mode).
* `remotepath` - (Optional) The remote path (for SSH mode).
* `direction` - (Optional) Sync direction: PUSH or PULL. Valid values: `PUSH`, `PULL`. Default: `PUSH`.
* `enabled` - (Optional) Whether the rsync task is enabled. Default: `true`.
* `desc` - (Optional) A description for the rsync task.
* `schedule_minute` - (Optional) Cron schedule minute field. Default: `0`.
* `schedule_hour` - (Optional) Cron schedule hour field. Default: `0`.
* `schedule_dom` - (Optional) Cron schedule day-of-month field. Default: `*`.
* `schedule_month` - (Optional) Cron schedule month field. Default: `*`.
* `schedule_dow` - (Optional) Cron schedule day-of-week field. Default: `*`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_rsync_task` resource.

## Import

The `truenas_rsync_task` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_rsync_task.example 1
```
