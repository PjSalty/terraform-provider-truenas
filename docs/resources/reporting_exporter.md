---
page_title: "truenas_reporting_exporter Resource - terraform-provider-truenas"
subcategory: "Alerts & Monitoring"
description: |-
  Manages a reporting exporter (e.g. Graphite) on TrueNAS SCALE.
---

# truenas_reporting_exporter (Resource)

Manages a reporting exporter (e.g. Graphite) on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_reporting_exporter" "example" {
  name    = "prometheus"
  type    = "PROMETHEUS"
  enabled = true
  attributes_json = jsonencode({
    port = 9100
  })
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) User-defined name of the exporter.
* `attributes_json` - (Required) Exporter-specific attributes as a JSON object, including `exporter_type`. Example: jsonencode({exporter_type="GRAPHITE", destination_ip="1.2.3.4", destination_port=2003, namespace="truenas"}).
* `enabled` - (Optional) Whether this exporter is enabled. Default: `true`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_reporting_exporter` resource.

## Import

The `truenas_reporting_exporter` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_reporting_exporter.example 1
```
