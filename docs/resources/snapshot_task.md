---
page_title: "truenas_snapshot_task Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages a periodic ZFS snapshot task on TrueNAS SCALE.
---

# truenas_snapshot_task (Resource)

Manages a periodic ZFS snapshot task on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_dataset" "snap_source" {
  pool = "tank"
  name = "snap_source"
}

resource "truenas_snapshot_task" "example" {
  dataset        = truenas_dataset.snap_source.id
  recursive      = true
  lifetime_value = 2
  lifetime_unit  = "WEEK"
  naming_schema  = "auto-%Y-%m-%d_%H-%M"
  schedule {
    minute = "0"
    hour   = "*/4"
  }
  enabled = true
}
```

## Argument Reference

The following arguments are supported:

* `dataset` - (Required) The dataset to snapshot (e.g., tank/data).
* `recursive` - (Optional) Whether to recursively snapshot child datasets. Default: `false`.
* `lifetime_value` - (Optional) How long to keep snapshots (numeric value). Default: `2`.
* `lifetime_unit` - (Optional) Lifetime unit (HOUR, DAY, WEEK, MONTH, YEAR). Valid values: `HOUR`, `DAY`, `WEEK`, `MONTH`, `YEAR`. Default: `WEEK`.
* `naming_schema` - (Optional) Naming schema for snapshots (e.g., auto-%Y-%m-%d_%H-%M). Default: `auto-%Y-%m-%d_%H-%M`.
* `enabled` - (Optional) Whether the task is enabled. Default: `true`.
* `allow_empty` - (Optional) Whether to create snapshots even if there are no changes. Default: `true`.
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

* `id` - The unique identifier of the `truenas_snapshot_task` resource.

## Import

The `truenas_snapshot_task` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_snapshot_task.example 1
```
