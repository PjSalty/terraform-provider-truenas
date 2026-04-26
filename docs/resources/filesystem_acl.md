---
page_title: "truenas_filesystem_acl Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages POSIX ACLs on TrueNAS SCALE filesystem paths.
---

# truenas_filesystem_acl (Resource)

Manages POSIX ACLs on TrueNAS SCALE filesystem paths.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_dataset" "acl_data" {
  pool = "tank"
  name = "acl_data"
}

resource "truenas_filesystem_acl" "example" {
  path    = "/mnt/tank/acl_data"
  acltype = "NFS4"

  dacl = [
    {
      tag          = "owner@"
      type         = "ALLOW"
      perm_read    = true
      perm_write   = true
      perm_execute = true
      default      = false
    },
    {
      tag          = "group@"
      type         = "ALLOW"
      perm_read    = true
      perm_write   = true
      perm_execute = true
      default      = false
    },
    {
      tag          = "everyone@"
      type         = "ALLOW"
      perm_read    = true
      perm_write   = false
      perm_execute = true
      default      = false
    },
  ]
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The filesystem path to manage ACLs on (e.g., /mnt/pool/dataset). Changing this attribute forces a new resource to be created.
* `dacl` - (Required) List of ACL entries.
* `acltype` - (Optional) The ACL type (POSIX1E or NFS4). Valid values: `POSIX1E`, `NFS4`. Default: `POSIX1E`.
* `uid` - (Optional) The owner UID. Default: `0`.
* `gid` - (Optional) The owner GID. Default: `0`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_filesystem_acl` resource.

## Import

The `truenas_filesystem_acl` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
# Filesystem ACLs are imported by the target path
terraform import truenas_filesystem_acl.example /mnt/tank/acl_data
```
