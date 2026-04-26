---
page_title: "truenas_filesystem_acl_template Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages a filesystem ACL template on TrueNAS SCALE.
---

# truenas_filesystem_acl_template (Resource)

Manages a filesystem ACL template on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_filesystem_acl_template" "example" {
  name    = "restricted-share"
  acltype = "NFS4"
  acl_json = jsonencode([
    {
      tag   = "owner@"
      type  = "ALLOW"
      perms = { BASIC = "FULL_CONTROL" }
      flags = { BASIC = "INHERIT" }
    }
  ])
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Human-readable template name.
* `acltype` - (Required) ACL type this template provides: NFS4 or POSIX1E. Valid values: `NFS4`, `POSIX1E`. Changing this attribute forces a new resource to be created.
* `acl_json` - (Required) ACL entries as a JSON array (see resource docs).
* `comment` - (Optional) Optional comment. Default: ``.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_filesystem_acl_template` resource.
* `builtin` - True if this is a TrueNAS built-in template (read-only).

## Import

The `truenas_filesystem_acl_template` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_filesystem_acl_template.example 1
```
