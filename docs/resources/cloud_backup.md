---
page_title: "truenas_cloud_backup Resource - terraform-provider-truenas"
subcategory: "Auth & Integration"
description: |-
  Manages a cloud backup (restic) task on TrueNAS SCALE.
---

# truenas_cloud_backup (Resource)

Manages a cloud backup (restic) task on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_cloud_backup" "example" {
  path        = "/mnt/tank/backup"
  credentials = 1 # numeric ID of a cloud credential
  password    = "restic-repo-password"
  keep_last   = 30
  enabled     = true
  attributes_json = jsonencode({
    bucket = "example-restic"
    folder = "truenas"
  })
  schedule {
    minute = "0"
    hour   = "5"
  }
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) Local path to back up (must begin with /mnt or /dev/zvol).
* `credentials` - (Required) ID of the cloud credential to use for this task.
* `attributes_json` - (Required) Provider-specific attributes as a JSON object (e.g. jsonencode({bucket="b", region="us-east-1"})).
* `password` - (Required) Password for the remote restic repository. Marked sensitive.
* `keep_last` - (Required) How many of the most recent backup snapshots to keep after each backup.
* `description` - (Optional) Human-readable name for the backup task. Default: ``.
* `pre_script` - (Optional) Bash script to run immediately before each backup. Default: ``.
* `post_script` - (Optional) Bash script to run immediately after each successful backup. Default: ``.
* `snapshot` - (Optional) Create a temporary snapshot of the dataset before each backup. Default: `false`.
* `include` - (Optional) Paths to pass to restic backup --include.
* `exclude` - (Optional) Paths to pass to restic backup --exclude.
* `args` - (Optional) Additional args (slated for removal upstream). Default: ``.
* `enabled` - (Optional) Whether this task is enabled. Default: `true`.
* `transfer_setting` - (Optional) One of DEFAULT, PERFORMANCE, FAST_STORAGE. Valid values: `DEFAULT`, `PERFORMANCE`, `FAST_STORAGE`. Default: `DEFAULT`.
* `schedule_minute` - (Optional) Default: `00`.
* `schedule_hour` - (Optional) Default: `*`.
* `schedule_dom` - (Optional) Default: `*`.
* `schedule_month` - (Optional) Default: `*`.
* `schedule_dow` - (Optional) Default: `*`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_cloud_backup` resource.

## Import

The `truenas_cloud_backup` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_cloud_backup.example 1
```
