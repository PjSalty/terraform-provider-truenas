---
page_title: "truenas_replication Resource - terraform-provider-truenas"
subcategory: "Scheduling"
description: |-
  Manages a ZFS replication task on TrueNAS SCALE.
---

# truenas_replication (Resource)

Manages a ZFS replication task on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
# Push tank/data to another system after each run of periodic snapshot task 3,
# at most once every two hours. ssh_credentials is the ID of an
# SSH_CREDENTIALS keychain credential.
resource "truenas_replication" "example" {
  name             = "tank-offsite"
  direction        = "PUSH"
  transport        = "SSH"
  ssh_credentials  = 12
  source_datasets  = ["tank/data"]
  target_dataset   = "backup/tank"
  recursive        = true
  enabled          = true
  auto             = true
  retention_policy = "NONE"

  periodic_snapshot_tasks = [3]
  hold_pending_snapshots  = true
  restrict_schedule       = [{ hour = "*/2" }]
}

# Pull it back every six hours on the task's own schedule.
resource "truenas_replication" "scheduled" {
  name             = "offsite-pull"
  direction        = "PULL"
  transport        = "SSH"
  ssh_credentials  = 12
  source_datasets  = ["backup/tank"]
  target_dataset   = "tank/restore"
  auto             = true
  retention_policy = "NONE"
  naming_schema    = ["auto-%Y-%m-%d_%H-%M"]

  schedule = [{ hour = "*/6" }]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the replication task.
* `direction` - (Required) Replication direction (PUSH or PULL). Valid values: `PUSH`, `PULL`.
* `source_datasets` - (Required) List of source dataset paths.
* `target_dataset` - (Required) Target dataset path.
* `transport` - (Optional) Transport type (SSH, SSH+NETCAT, LOCAL). Valid values: `SSH`, `SSH+NETCAT`, `LOCAL`, `LEGACY`. Default: `LOCAL`.
* `recursive` - (Optional) Whether to recursively replicate child datasets. Default: `false`.
* `auto` - (Optional) Whether the task runs on its own. TrueNAS accepts `auto = true` only with a `schedule`, or, for a push task, with `periodic_snapshot_tasks`. Set `auto = false` for a task that is started by hand. Default: `true`.
* `enabled` - (Optional) Whether the replication task is enabled. Default: `true`.
* `retention_policy` - (Optional) Snapshot retention policy (SOURCE, CUSTOM, NONE). Valid values: `SOURCE`, `CUSTOM`, `NONE`. Default: `SOURCE`.
* `lifetime_value` - (Optional) Lifetime value for CUSTOM retention policy.
* `lifetime_unit` - (Optional) Lifetime unit for CUSTOM retention (HOUR, DAY, WEEK, MONTH, YEAR). Valid values: `HOUR`, `DAY`, `WEEK`, `MONTH`, `YEAR`.
* `ssh_credentials` - (Optional) SSH credentials ID for remote replication.
* `naming_schema` - (Optional) Naming schema for matching snapshots on the source (for pull replication).
* `also_include_naming_schema` - (Optional) Naming schema for snapshots to include (for push replication without periodic snapshot tasks).
* `periodic_snapshot_tasks` - (Optional) IDs of the periodic snapshot tasks a push replication is bound to. The task replicates their snapshots and, with `auto = true`, runs after each of them. Left out, the current binding is kept. Set to `[]` to unbind every task.
* `hold_pending_snapshots` - (Optional) Keep source snapshots that have not been replicated yet, so that retention cannot delete them after a failed run. Push replication only. Left out, the current value is kept; TrueNAS defaults to false.
* `schedule` - (Optional) When the task runs on its own: a list with one entry, see [below](#schedule). Left out, the current schedule is kept. Set to `[]` for no schedule, which TrueNAS requires when `auto = false`.
* `restrict_schedule` - (Optional) Limits when a task bound to `periodic_snapshot_tasks` runs: a list with one entry, see [below](#schedule). For example, snapshot tasks can run every 15 minutes while the replication runs once an hour. Left out, the current value is kept. Set to `[]` for no restriction.
* `only_matching_schedule` - (Optional) Replicate only the snapshots that match `schedule` or `restrict_schedule`. TrueNAS accepts true only when `schedule` is set. Left out, the current value is kept; TrueNAS defaults to false.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Schedule

The entry of `schedule` and of `restrict_schedule` supports the fields below. A field that is left out keeps its current value, or takes the TrueNAS default on a new entry.

* `minute` - (Optional) Cron minute field. TrueNAS defaults to `00`.
* `hour` - (Optional) Cron hour field. TrueNAS defaults to `*`.
* `dom` - (Optional) Cron day-of-month field. TrueNAS defaults to `*`.
* `month` - (Optional) Cron month field. TrueNAS defaults to `*`.
* `dow` - (Optional) Cron day-of-week field, `1` (Monday) to `7`. TrueNAS defaults to `*`.
* `begin` - (Optional) Start of the daily window in which the schedule applies, as HH:MM. TrueNAS defaults to `00:00`.
* `end` - (Optional) End of the daily window, as HH:MM. TrueNAS defaults to `23:59`.

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_replication` resource.

## Import

The `truenas_replication` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_replication.example 1
```
