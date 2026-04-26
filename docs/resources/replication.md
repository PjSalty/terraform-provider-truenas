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
resource "truenas_replication" "example" {
  name             = "tank-offsite"
  direction        = "PUSH"
  transport        = "SSH"
  source_datasets  = ["tank/data"]
  target_dataset   = "backup/tank"
  recursive        = true
  enabled          = true
  auto             = true
  retention_policy = "NONE"
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
* `auto` - (Optional) Whether to run the replication automatically on schedule. Default: `true`.
* `enabled` - (Optional) Whether the replication task is enabled. Default: `true`.
* `retention_policy` - (Optional) Snapshot retention policy (SOURCE, CUSTOM, NONE). Valid values: `SOURCE`, `CUSTOM`, `NONE`. Default: `SOURCE`.
* `lifetime_value` - (Optional) Lifetime value for CUSTOM retention policy.
* `lifetime_unit` - (Optional) Lifetime unit for CUSTOM retention (HOUR, DAY, WEEK, MONTH, YEAR). Valid values: `HOUR`, `DAY`, `WEEK`, `MONTH`, `YEAR`.
* `ssh_credentials` - (Optional) SSH credentials ID for remote replication.
* `naming_schema` - (Optional) Naming schema for matching snapshots on the source (for pull replication).
* `also_include_naming_schema` - (Optional) Naming schema for snapshots to include (for push replication without periodic snapshot tasks).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_replication` resource.

## Import

The `truenas_replication` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_replication.example 1
```
