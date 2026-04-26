---
page_title: "truenas_nvmet_host_subsys Resource - terraform-provider-truenas"
subcategory: "NVMe-oF"
description: |-
  Manages an NVMe-oF host-to-subsystem authorization. This grants a host (initiator NQN) access to a subsystem (target). Both host_id and subsys_id require replacement if changed.
---

# truenas_nvmet_host_subsys (Resource)

Manages an NVMe-oF host-to-subsystem authorization. This grants a host (initiator NQN) access to a subsystem (target). Both host_id and subsys_id require replacement if changed.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_nvmet_host" "h" {
  hostnqn = "nqn.2014-08.org.nvmexpress:uuid:22222222-2222-2222-2222-222222222222"
}

resource "truenas_nvmet_subsys" "s" {
  name = "hostsubsysexample"
}

resource "truenas_nvmet_host_subsys" "example" {
  host_id   = tonumber(truenas_nvmet_host.h.id)
  subsys_id = tonumber(truenas_nvmet_subsys.s.id)
}
```

## Argument Reference

The following arguments are supported:

* `host_id` - (Required) ID of the NVMe-oF host to authorize. Changing this attribute forces a new resource to be created.
* `subsys_id` - (Required) ID of the NVMe-oF subsystem to grant access to. Changing this attribute forces a new resource to be created.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_nvmet_host_subsys` resource.

## Import

The `truenas_nvmet_host_subsys` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_nvmet_host_subsys.example 1
```
