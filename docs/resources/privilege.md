---
page_title: "truenas_privilege Resource - terraform-provider-truenas"
subcategory: "Users & RBAC"
description: |-
  Manages a TrueNAS RBAC privilege — a named grant of roles to one or more local and/or directory-service groups.
---

# truenas_privilege (Resource)

Manages a TrueNAS RBAC privilege — a named grant of roles to one or more local and/or directory-service groups.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_privilege" "example" {
  name         = "zfs-operators"
  local_groups = [] # filled in with numeric group IDs
  ds_groups    = []
  roles        = ["SHARING_NFS_WRITE", "SHARING_SMB_WRITE"]
  web_shell    = false
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Display name of the privilege (must be unique).
* `local_groups` - (Optional) List of local group GIDs granted by this privilege.
* `ds_groups` - (Optional) List of directory-service group identifiers (GIDs or SID strings) granted by this privilege.
* `roles` - (Optional) List of role names included in this privilege (e.g. READONLY_ADMIN, FULL_ADMIN).
* `web_shell` - (Optional) Whether holders of this privilege may access the TrueNAS web shell. Default: `false`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_privilege` resource.

## Import

The `truenas_privilege` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_privilege.example 1
```
