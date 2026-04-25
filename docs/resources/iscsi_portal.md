---
page_title: "truenas_iscsi_portal Resource - terraform-provider-truenas"
subcategory: "iSCSI"
description: |-
  Manages an iSCSI portal on TrueNAS SCALE.
---

# truenas_iscsi_portal (Resource)

Manages an iSCSI portal on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_iscsi_portal" "example" {
  comment = "Primary iSCSI portal"

  listen {
    ip   = "0.0.0.0"
    port = 3260
  }
}
```

## Argument Reference

The following arguments are supported:

* `listen` - (Required) Listen addresses for the portal.
* `comment` - (Optional) A comment for the portal.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_iscsi_portal` resource.
* `tag` - The portal group tag.

## Import

The `truenas_iscsi_portal` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_iscsi_portal.example 1
```
