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

### Basic

```terraform
resource "truenas_keychain_credential" "example" {
  name = "aws-s3-backup"
  type = "S3_BUCKET"
  attributes = {
    access_key_id     = "AKIAEXAMPLE"
    secret_access_key = "examplesecret"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the keychain credential.
* `type` - (Required) The credential type: SSH_KEY_PAIR or SSH_CREDENTIALS. Valid values: `SSH_KEY_PAIR`, `SSH_CREDENTIALS`. Changing this attribute forces a new resource to be created.
* `attributes` - (Required) The credential attributes. For SSH_KEY_PAIR: private_key, public_key. For SSH_CREDENTIALS: host, username, private_key, connect_timeout. Marked sensitive.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_keychain_credential` resource.

## Import

The `truenas_keychain_credential` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_keychain_credential.example 1
```
