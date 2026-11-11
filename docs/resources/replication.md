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
* `readonly` - (Optional) What the task does with the `readonly` property of the destination datasets: `SET` turns it on after each run, `REQUIRE` refuses to replicate into a dataset that has it off, and `IGNORE` leaves it alone. Valid values: `SET`, `REQUIRE`, `IGNORE`. Left out, the current value is kept; TrueNAS defaults to `SET`.
* `allow_from_scratch` - (Optional) When no snapshot on the destination matches the source, destroy the destination's snapshots and replicate everything again. Left out, the current value is kept; TrueNAS defaults to false.
* `retries` - (Optional) Number of attempts before a run counts as failed. Left out, the current value is kept; TrueNAS defaults to 5.
* `exclude` - (Optional) Child datasets to leave out of a recursive replication. Left out, the current list is kept. Set to `[]` to exclude nothing.
* `properties` - (Optional) Send dataset properties along with the snapshots. Left out, the current value is kept; TrueNAS defaults to true.
* `properties_exclude` - (Optional) Dataset properties that are not sent. Left out, the current list is kept. Set to `[]` to send them all.
* `properties_override` - (Optional) Dataset properties to set on the destination in place of the source's values. Left out, the current map is kept. Set to `{}` to override nothing.
* `replicate` - (Optional) Full filesystem replication: every child dataset, snapshot and property. TrueNAS then requires `recursive` and `properties` to be true and `exclude` to be empty. Left out, the current value is kept; TrueNAS defaults to false.
* `compression` - (Optional) Compression of the SSH stream: `LZ4`, `PIGZ` or `PLZIP`, or `""` for none. SSH transport only. Valid values: ``, `LZ4`, `PIGZ`, `PLZIP`. Left out, the current value is kept.
* `speed_limit` - (Optional) Limit of the SSH stream in bytes per second, or `0` for no limit. SSH transport only. Left out, the current value is kept.
* `large_block` - (Optional) Allow blocks larger than 128 KiB in the send stream. Left out, the current value is kept; TrueNAS defaults to true.
* `embed` - (Optional) Allow embedded blocks in the send stream. Left out, the current value is kept; TrueNAS defaults to false.
* `compressed` - (Optional) Send blocks as they are compressed on disk. Left out, the current value is kept; TrueNAS defaults to true.
* `sudo` - (Optional) Run `zfs` on the remote system through passwordless `sudo`. SSH and SSH+NETCAT transports only. Left out, the current value is kept; TrueNAS defaults to false.
* `logging_level` - (Optional) Log level of the task's runs: `DEBUG`, `INFO`, `WARNING` or `ERROR`, or `""` for the TrueNAS default. Valid values: ``, `DEBUG`, `INFO`, `WARNING`, `ERROR`. Left out, the current value is kept.
* `name_regex` - (Optional) Replicate every snapshot whose name matches this regular expression, or `""` for none. TrueNAS does not accept it together with a naming schema or with `periodic_snapshot_tasks`. Left out, the current value is kept.
* `lifetimes` - (Optional) Retention periods for `retention_policy = "CUSTOM"`, each for the snapshots that match its schedule, see [below](#lifetimes). Left out, the current list is kept. Set to `[]` for none.
* `encryption` - (Optional) Encrypt the replicated datasets on the destination. TrueNAS then requires either `encryption_inherit = true` or `encryption_key`, `encryption_key_format` and `encryption_key_location`. Left out, the current value is kept; TrueNAS defaults to false.
* `encryption_inherit` - (Optional) Take the encryption settings from the destination's parent dataset. Left out, the current value is kept; it is false where TrueNAS has none.
* `encryption_key` - (Optional, Sensitive) Key or passphrase for the replicated datasets, or `""` for none. Left out, the current value is kept.
* `encryption_key_format` - (Optional) Form of `encryption_key`: `HEX` or `PASSPHRASE`, or `""` for none. Valid values: ``, `HEX`, `PASSPHRASE`. Left out, the current value is kept.
* `encryption_key_location` - (Optional) Path where the encryption key is stored, or `""` for none. Left out, the current value is kept.
* `netcat_active_side` - (Optional) Which side opens the netcat connection of an `SSH+NETCAT` transport: `LOCAL` or `REMOTE`, or `""` for none. TrueNAS requires it for `SSH+NETCAT` and accepts the netcat attributes for no other transport. Valid values: ``, `LOCAL`, `REMOTE`. Left out, the current value is kept.
* `netcat_active_side_listen_address` - (Optional) Address the active side listens on, or `""` for none. `SSH+NETCAT` only. Left out, the current value is kept.
* `netcat_active_side_port_min` - (Optional) Lowest port the active side may use, or `0` for none. `SSH+NETCAT` only. Left out, the current value is kept.
* `netcat_active_side_port_max` - (Optional) Highest port the active side may use, or `0` for none. `SSH+NETCAT` only. Left out, the current value is kept.
* `netcat_passive_side_connect_address` - (Optional) Address the passive side connects to, or `""` for none. `SSH+NETCAT` only. Left out, the current value is kept.
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

### Lifetimes

Each entry of `lifetimes` supports:

* `schedule` - (Required) Cron schedule of the snapshots this period applies to: an object with `minute`, `hour`, `dom`, `month` and `dow`, all required.
* `lifetime_value` - (Required) How many units the matching snapshots are kept.
* `lifetime_unit` - (Required) Unit of `lifetime_value`. Valid values: `HOUR`, `DAY`, `WEEK`, `MONTH`, `YEAR`.

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
