---
page_title: "truenas_alert_service Resource - terraform-provider-truenas"
subcategory: "Alerts & Monitoring"
description: |-
  Manages an alert service on TrueNAS SCALE (email, Pushover, Slack, etc.).
---

# truenas_alert_service (Resource)

Manages an alert service on TrueNAS SCALE (email, Pushover, Slack, etc.).

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_alert_service" "example" {
  name    = "ops-email"
  type    = "Mail"
  enabled = true
  level   = "WARNING"

  settings_json = jsonencode({
    email = "ops@example.com"
  })
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the alert service.
* `type` - (Required) The alert service type (Mail, PushOver, Slack, PagerDuty, etc.).
* `settings_json` - (Required) Service-specific settings as a JSON string. The structure depends on the service type. Marked sensitive.
* `enabled` - (Optional) Whether the alert service is enabled. Default: `true`.
* `level` - (Optional) Minimum alert level (INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY). Valid values: `INFO`, `NOTICE`, `WARNING`, `ERROR`, `CRITICAL`, `ALERT`, `EMERGENCY`. Default: `WARNING`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_alert_service` resource.

## Import

The `truenas_alert_service` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_alert_service.example 1
```
