---
page_title: "truenas_tunable Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages a kernel tunable (sysctl, udev, or ZFS parameter) on TrueNAS SCALE.
---

# truenas_tunable (Resource)

Manages a kernel tunable (sysctl, udev, or ZFS parameter) on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_tunable" "example" {
  type    = "SYSCTL"
  var     = "vfs.zfs.arc_max"
  value   = "17179869184" # 16 GiB
  comment = "Cap ARC at 16 GiB"
  enabled = true
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The type of tunable: SYSCTL, UDEV, or ZFS. Valid values: `SYSCTL`, `UDEV`, `ZFS`. Changing this attribute forces a new resource to be created.
* `var` - (Required) The variable name (e.g., net.ipv4.ip_forward). Changing this attribute forces a new resource to be created.
* `value` - (Required) The value to set.
* `comment` - (Optional) A comment describing the tunable. Default: ``.
* `enabled` - (Optional) Whether the tunable is enabled. Default: `true`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_tunable` resource.

## Import

The `truenas_tunable` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_tunable.example 1
```
