---
page_title: "truenas_scrub_task Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages a ZFS pool scrub schedule on TrueNAS SCALE.
---

# truenas_scrub_task (Resource)

Manages a ZFS pool scrub schedule on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
data "truenas_pool" "tank" {
  name = "tank"
}

resource "truenas_scrub_task" "example" {
  pool        = data.truenas_pool.tank.id
  threshold   = 35
  description = "Weekly scrub"
  enabled     = true
  schedule {
    minute = "0"
    hour   = "0"
    dow    = "0" # Sunday
  }
}
```

## Argument Reference

The following arguments are supported:

* `pool` - (Required) The pool ID to scrub.
* `threshold` - (Optional) Number of days between scrubs (threshold). Default: `35`.
* `description` - (Optional) A description for the scrub task. Default: ``.
* `enabled` - (Optional) Whether the scrub task is enabled. Default: `true`.
* `schedule_minute` - (Optional) Cron schedule minute field. Default: `00`.
* `schedule_hour` - (Optional) Cron schedule hour field. Default: `00`.
* `schedule_dom` - (Optional) Cron schedule day-of-month field. Default: `*`.
* `schedule_month` - (Optional) Cron schedule month field. Default: `*`.
* `schedule_dow` - (Optional) Cron schedule day-of-week field. Default: `7`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_scrub_task` resource.
* `pool_name` - The pool name (read-only, populated by API).

## Import

The `truenas_scrub_task` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_scrub_task.example 1
```
