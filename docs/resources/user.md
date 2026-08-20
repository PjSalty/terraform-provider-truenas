---
page_title: "truenas_user Resource - terraform-provider-truenas"
subcategory: "Users & RBAC"
description: |-
  Manages a local user on TrueNAS SCALE.
---

# truenas_user (Resource)

Manages a local user on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_group" "admins" {
  name = "admins"
}

resource "truenas_user" "example" {
  username          = "alice"
  full_name         = "Alice Example"
  group             = tonumber(truenas_group.admins.id)
  password          = "ChangeMe!2026"
  shell             = "/usr/bin/bash"
  home              = "/var/empty"
  home_mode         = "0755"
  sshpubkey         = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... alice@example"
  password_disabled = false
  locked            = false
}
```

### Passwordless service account

An account that only owns files, such as an NFS `mapall` user, needs no
password login:

```hcl
resource "truenas_user" "nfs_mapall" {
  username          = "nfsmap"
  full_name         = "NFS mapall account"
  password_disabled = true
  shell             = "/usr/sbin/nologin"
  home              = "/var/empty"
  smb               = false
}
```

## Argument Reference

The following arguments are supported:

* `username` - (Required) The login name of the user. Changing this attribute forces a new resource to be created.
* `full_name` - (Required) The full (display) name of the user.
* `password` - (Optional) The password for the user. Marked sensitive. Omit it
  and set `password_disabled = true` for an account that should have no
  password login. Exactly one of the two is required on create.
* `webshare` - (Optional) Allow this account to access WebShare shares.
  Requires TrueNAS 26.0 or newer; setting it on an older server is an error.
  Setting it adds the user to the builtin `truenas_webshare` group. On TrueNAS
  below 26.0 the field does not exist and always reads as `false`.
* `password_disabled` - (Optional) Disable password authentication for this
  user. Default: `false`. Cannot be combined with `password`, and TrueNAS
  rejects it for SMB users because Samba needs a password to derive the NT
  hash. Turning it back off requires setting `password` in the same change.
* `uid` - (Optional) The UNIX UID for the user. If not set, TrueNAS will assign one.
* `email` - (Optional) Email address of the user. Default: ``.
* `group` - (Optional) The primary group ID. If not specified and group_create is true, a group matching the username will be created.
* `group_create` - (Optional) Whether to create a new primary group for the user. Default: `true`.
* `groups` - (Optional) Auxiliary group IDs. This is an unordered set; the order of IDs is not significant.
* `home` - (Optional) Home directory path. Must begin with /mnt or be /var/empty. Default: `/var/empty`.
* `shell` - (Optional) Login shell (e.g., /usr/sbin/nologin, /bin/bash). Default: `/usr/sbin/nologin`.
* `locked` - (Optional) Whether the user account is locked. Default: `false`.
* `smb` - (Optional) Whether the user should have Samba authentication enabled. Default: `false`.
* `sshpubkey` - (Optional) SSH public key for the user. Default: ``.
* `sudo_commands` - (Optional) List of sudo commands the user is allowed to run.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_user` resource.

## Import

The `truenas_user` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_user.example 123
```
