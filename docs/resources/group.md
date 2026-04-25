---
page_title: "truenas_group Resource - terraform-provider-truenas"
subcategory: "Users & RBAC"
description: |-
  Manages a local group on TrueNAS SCALE.
---

# truenas_group (Resource)

Manages a local group on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_group" "example" {
  name                   = "developers"
  sudo_commands_nopasswd = ["/usr/bin/zfs"]
  smb                    = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the group.
* `gid` - (Optional) The GID for the group. If not set, TrueNAS will assign one.
* `smb` - (Optional) Whether the group should be mapped to a Samba group. Default: `false`.
* `sudo_commands` - (Optional) List of sudo commands the group members are allowed to run.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_group` resource.

## Import

The `truenas_group` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_group.example 123
```
