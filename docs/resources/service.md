---
page_title: "truenas_service Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages a TrueNAS service (enable/disable and start/stop).
---

# truenas_service (Resource)

Manages a TrueNAS service (enable/disable and start/stop).

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_service" "ssh" {
  service = "ssh"
  enable  = true
  state   = "RUNNING"
}
```

## Argument Reference

The following arguments are supported:

* `service` - (Required) The service name (e.g., nfs, cifs, ssh, iscsitarget, snmp, ftp, ups). Changing this attribute forces a new resource to be created.
* `enable` - (Optional) Whether the service is enabled to start on boot. Default: `false`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_service` resource.
* `state` - The current state of the service (RUNNING or STOPPED).

## Import

The `truenas_service` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_service.ssh ssh
```
