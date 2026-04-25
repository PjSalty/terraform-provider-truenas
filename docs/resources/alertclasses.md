---
page_title: "truenas_alertclasses Resource - terraform-provider-truenas"
subcategory: "Alerts & Monitoring"
description: |-
  Manages the singleton alert class severity/notification-policy configuration on TrueNAS SCALE.
---

# truenas_alertclasses (Resource)

Manages the singleton alert class severity/notification-policy configuration on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
# Singleton: global alert-class policy overrides.
resource "truenas_alertclasses" "this" {
  classes = {
    ZpoolCapacityWarning = {
      level  = "WARNING"
      policy = "IMMEDIATELY"
    }
    ZpoolCapacityCritical = {
      level  = "CRITICAL"
      policy = "IMMEDIATELY"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `classes` - (Required) Map of alert class name to configuration.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_alertclasses` resource.

## Import

The `truenas_alertclasses` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_alertclasses.this singleton
```
