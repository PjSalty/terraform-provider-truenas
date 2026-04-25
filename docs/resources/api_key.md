---
page_title: "truenas_api_key Resource - terraform-provider-truenas"
subcategory: "Users & RBAC"
description: |-
  Manages an API key on TrueNAS SCALE.
---

# truenas_api_key (Resource)

Manages an API key on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_user" "svc" {
  username  = "svcterraform"
  full_name = "Terraform Service Account"
  group     = 0
  password  = "rotated-by-terraform"
}

resource "truenas_api_key" "example" {
  name     = "terraform-ci"
  username = truenas_user.svc.username
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the API key.
* `username` - (Required) The username this API key is associated with. Changing this attribute forces a new resource to be created.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_api_key` resource.
* `key` - The API key value. Only available on creation. Marked sensitive.

## Import

The `truenas_api_key` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_api_key.example 1
```
