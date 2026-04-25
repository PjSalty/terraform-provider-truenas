---
page_title: "truenas_nvmet_global Resource - terraform-provider-truenas"
subcategory: "NVMe-oF"
description: |-
  Manages the NVMe-oF global configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist. Delete does not reset the remote configuration; it only removes the resource from Terraform state.
---

# truenas_nvmet_global (Resource)

Manages the NVMe-oF global configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist. Delete does not reset the remote configuration; it only removes the resource from Terraform state.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_nvmet_global" "this" {
  basenqn        = "nqn.2011-06.com.truenas"
  ana            = false
  rdma           = false
  xport_referral = true
}
```

## Argument Reference

The following arguments are supported:

* `basenqn` - (Optional) NQN used as the prefix when creating subsystems without an explicit subnqn.
* `kernel` - (Optional) Use the kernel NVMe-oF backend.
* `ana` - (Optional) Enable Asymmetric Namespace Access (ANA).
* `rdma` - (Optional) Enable RDMA transport (Enterprise + RDMA-capable hardware only).
* `xport_referral` - (Optional) Generate cross-port referrals for ports on this TrueNAS.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_nvmet_global` resource.

## Import

The `truenas_nvmet_global` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_nvmet_global.this singleton
```
