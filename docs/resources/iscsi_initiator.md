---
page_title: "truenas_iscsi_initiator Resource - terraform-provider-truenas"
subcategory: "iSCSI"
description: |-
  Manages an iSCSI authorized initiator group on TrueNAS SCALE.
---

# truenas_iscsi_initiator (Resource)

Manages an iSCSI authorized initiator group on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_iscsi_initiator" "example" {
  comment = "Allow trusted networks"
  # Leave initiators unset to allow ALL, or restrict:
  # initiators = ["iqn.2025-01.com.example:client1"]
}
```

## Argument Reference

The following arguments are supported:

* `initiators` - (Optional) List of initiator IQNs allowed to connect. Empty list allows all initiators.
* `comment` - (Optional) A comment for the initiator group.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_iscsi_initiator` resource.

## Import

The `truenas_iscsi_initiator` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_iscsi_initiator.example 1
```
