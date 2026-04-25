---
page_title: "truenas_cloud_sync Resource - terraform-provider-truenas"
subcategory: "Auth & Integration"
description: |-
  Manages a cloud sync task on TrueNAS SCALE.
---

# truenas_cloud_sync (Resource)

Manages a cloud sync task on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_cloud_sync" "example" {
  description   = "Offsite backup to S3"
  path          = "/mnt/tank/backup"
  credentials   = 1 # ID of a cloud credential keychain entry
  direction     = "PUSH"
  transfer_mode = "SYNC"
  enabled       = true
  attributes_json = jsonencode({
    bucket = "example-backup"
    folder = "truenas"
  })
  schedule {
    minute = "0"
    hour   = "4"
  }
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The local path to sync (e.g., /mnt/tank/backup).
* `credentials` - (Required) The ID of the cloud credential to use.
* `direction` - (Required) Sync direction: PUSH or PULL. Valid values: `PUSH`, `PULL`.
* `transfer_mode` - (Required) Transfer mode: SYNC, COPY, or MOVE. Valid values: `SYNC`, `COPY`, `MOVE`.
* `description` - (Optional) A description for the cloud sync task.
* `enabled` - (Optional) Whether the cloud sync task is enabled. Default: `true`.
* `attributes_json` - (Optional) JSON-encoded provider-specific attributes (e.g. `{"bucket":"my-bucket","folder":"/backups"}`). Exact keys depend on the cloud credential type. Default: `{}`.
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

* `id` - The unique identifier of the `truenas_cloud_sync` resource.

## Import

The `truenas_cloud_sync` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_cloud_sync.example 1
```
