---
page_title: "truenas_nvmet_host Resource - terraform-provider-truenas"
subcategory: "NVMe-oF"
description: |-
  Manages an NVMe-oF host (initiator NQN) on TrueNAS SCALE.
---

# truenas_nvmet_host (Resource)

Manages an NVMe-oF host (initiator NQN) on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_nvmet_host" "example" {
  hostnqn = "nqn.2014-08.org.nvmexpress:uuid:11111111-1111-1111-1111-111111111111"
}
```

## Argument Reference

The following arguments are supported:

* `hostnqn` - (Required) NQN of the host that will connect to this TrueNAS.
* `dhchap_key` - (Optional) Secret the host must present when connecting. Marked sensitive.
* `dhchap_ctrl_key` - (Optional) Secret TrueNAS will present to the host (bi-directional auth). Marked sensitive.
* `dhchap_dhgroup` - (Optional) Diffie-Hellman group used on top of CHAP (2048-BIT, 3072-BIT, 4096-BIT, 6144-BIT, 8192-BIT). Valid values: `2048-BIT`, `3072-BIT`, `4096-BIT`, `6144-BIT`, `8192-BIT`.
* `dhchap_hash` - (Optional) HMAC hash used for CHAP.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_nvmet_host` resource.

## Import

The `truenas_nvmet_host` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_nvmet_host.example 1
```
