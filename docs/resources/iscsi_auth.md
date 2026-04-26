---
page_title: "truenas_iscsi_auth Resource - terraform-provider-truenas"
subcategory: "iSCSI"
description: |-
  Manages an iSCSI CHAP authentication credential set.
---

# truenas_iscsi_auth (Resource)

Manages an iSCSI CHAP authentication credential set.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_iscsi_auth" "example" {
  tag    = 1
  user   = "chap-user"
  secret = "Sup3rSecretCHAP!"
  # peeruser + peersecret are only required for Mutual CHAP
}
```

## Argument Reference

The following arguments are supported:

* `tag` - (Required) Numeric tag used to associate this credential with iSCSI targets. Must be ≥ 0.
* `user` - (Required) Username for iSCSI CHAP authentication. Must be non-empty.
* `secret` - (Required) Password/secret for iSCSI CHAP authentication. Must be 12–16 characters per RFC 3720. Marked sensitive.
* `peeruser` - (Optional) Username for mutual CHAP authentication (optional). Default: ``.
* `peersecret` - (Optional) Password/secret for mutual CHAP authentication. When set, must be 12–16 characters per RFC 3720. Default: ``. Marked sensitive.
* `discovery_auth` - (Optional) Authentication method for target discovery: NONE, CHAP, CHAP_MUTUAL. Valid values: `NONE`, `CHAP`, `CHAP_MUTUAL`. Default: `NONE`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_iscsi_auth` resource.

## Import

The `truenas_iscsi_auth` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_iscsi_auth.example 1
```
