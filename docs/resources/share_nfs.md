---
page_title: "truenas_share_nfs Resource - terraform-provider-truenas"
subcategory: "Sharing"
description: |-
  Manages an NFS share on TrueNAS SCALE.
---

# truenas_share_nfs (Resource)

Manages an NFS share on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_dataset" "nfs_share" {
  pool = "tank"
  name = "nfs_share"
}

resource "truenas_share_nfs" "example" {
  path     = "/mnt/tank/nfs_share"
  comment  = "Example NFS export"
  enabled  = true
  networks = ["10.0.0.0/16"]

  maproot_user  = "root"
  maproot_group = "wheel"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path to share (e.g., /mnt/tank/data).
* `comment` - (Optional) A comment describing the share.
* `hosts` - (Optional) List of allowed hostnames or IP addresses. Empty means all hosts.
* `networks` - (Optional) List of allowed networks in CIDR notation.
* `readonly` - (Optional) Whether the share is read-only. Default: `false`.
* `maproot_user` - (Optional) Map root user to this user.
* `maproot_group` - (Optional) Map root group to this group.
* `mapall_user` - (Optional) Map all users to this user.
* `mapall_group` - (Optional) Map all groups to this group.
* `security` - (Optional) Security mechanisms (SYS, KRB5, KRB5I, KRB5P). Valid values: `SYS`, `KRB5`, `KRB5I`, `KRB5P`.
* `enabled` - (Optional) Whether the share is enabled. Default: `true`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_share_nfs` resource.

## Import

The `truenas_share_nfs` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_share_nfs.example 1
```
