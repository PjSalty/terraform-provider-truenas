---
page_title: "truenas_nvmet_subsys Resource - terraform-provider-truenas"
subcategory: "NVMe-oF"
description: |-
  Manages an NVMe-oF subsystem (target) on TrueNAS SCALE.
---

# truenas_nvmet_subsys (Resource)

Manages an NVMe-oF subsystem (target) on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_nvmet_subsys" "example" {
  name           = "examplesubsys"
  allow_any_host = false
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Human-readable name for the subsystem. If subnqn is not provided, this name is appended to the global basenqn.
* `subnqn` - (Optional) NVMe Qualified Name (NQN) for the subsystem. Auto-generated if not provided.
* `allow_any_host` - (Optional) Allow any host to access the storage in this subsystem (no access control).
* `pi_enable` - (Optional) Enable Protection Information (PI) for data integrity checking.
* `qid_max` - (Optional) Maximum number of queue IDs allowed for this subsystem.
* `ieee_oui` - (Optional) IEEE Organizationally Unique Identifier for the subsystem.
* `ana` - (Optional) Per-subsystem override of the global ANA setting. Leave unset to inherit global.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_nvmet_subsys` resource.
* `serial` - Serial number assigned to the subsystem (computed).

## Import

The `truenas_nvmet_subsys` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_nvmet_subsys.example 1
```
