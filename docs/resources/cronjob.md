---
page_title: "truenas_cronjob Resource - terraform-provider-truenas"
subcategory: "Scheduling"
description: |-
  Manages a cron job on TrueNAS SCALE.
---

# truenas_cronjob (Resource)

Manages a cron job on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_cronjob" "example" {
  command     = "/usr/bin/zpool scrub tank"
  description = "Weekly tank scrub"
  user        = "root"
  enabled     = true
  stdout      = true
  stderr      = true

  schedule {
    minute = "0"
    hour   = "3"
    dow    = "0"
  }
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) The user to run the cron job as.
* `command` - (Required) The command to execute.
* `description` - (Optional) A description for the cron job.
* `enabled` - (Optional) Whether the cron job is enabled. Default: `true`.
* `stdout` - (Optional) Whether to redirect stdout. Default: `true`.
* `stderr` - (Optional) Whether to redirect stderr. Default: `false`.
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

* `id` - The unique identifier of the `truenas_cronjob` resource.

## Import

The `truenas_cronjob` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_cronjob.example 1
```
