---
page_title: "truenas_keychain_credential Resource - terraform-provider-truenas"
subcategory: "Users & RBAC"
description: |-
  Manages a keychain credential (SSH key pair, SSH credentials) on TrueNAS SCALE.
---

# truenas_keychain_credential (Resource)

Manages a keychain credential (SSH key pair, SSH credentials) on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### SSH key pair and SSH connection

```terraform
resource "truenas_keychain_credential" "key" {
  name = "backup-key"
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = file("~/.ssh/truenas_backup")
    public_key  = file("~/.ssh/truenas_backup.pub")
  }
}

resource "truenas_keychain_credential" "backup" {
  name = "backup-host"
  type = "SSH_CREDENTIALS"
  attributes = {
    host            = "backup.example.com"
    port            = "22"
    username        = "root"
    private_key     = truenas_keychain_credential.key.id
    remote_host_key = "ssh-ed25519 AAAAC3Nza..."
    connect_timeout = "10"
  }
}
```

The `backup` credential can then be used by an SSH replication as
`ssh_credentials = tonumber(truenas_keychain_credential.backup.id)`, all in
one apply.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the keychain credential.
* `type` - (Required) The credential type: SSH_KEY_PAIR or SSH_CREDENTIALS. Valid values: `SSH_KEY_PAIR`, `SSH_CREDENTIALS`. Changing this attribute forces a new resource to be created.
* `attributes` - (Required) The credential attributes, as a map of strings. Marked sensitive.
  * For `SSH_KEY_PAIR`: `private_key` (the OpenSSH private key text) and `public_key`.
  * For `SSH_CREDENTIALS`: `host`, `port` (default 22), `username` (default `root`), `private_key` (the `id` of an `SSH_KEY_PAIR` credential), `remote_host_key`, and `connect_timeout` in seconds (default 10). `port`, `private_key` and `connect_timeout` must be whole numbers; the provider sends them to TrueNAS as integers. Set all six so the values read back match the configuration.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_keychain_credential` resource.

## Import

The `truenas_keychain_credential` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_keychain_credential.example 1
```
