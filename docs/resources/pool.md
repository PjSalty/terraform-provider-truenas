---
page_title: "truenas_pool Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages a ZFS pool on TrueNAS SCALE. Creation is asynchronous and the topology is passed through to the API as a raw JSON object to avoid modeling the deeply-nested discriminated-union schema.
---

# truenas_pool (Resource)

Manages a ZFS pool on TrueNAS SCALE. Creation is asynchronous and the topology is passed through to the API as a raw JSON object to avoid modeling the deeply-nested discriminated-union schema.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
# Creating a pool requires raw disk device names, which is only safe
# on fresh hardware. The topology is passed through as JSON to avoid
# modelling TrueNAS's deeply-nested vdev schema in HCL.
resource "truenas_pool" "example" {
  name = "tank"

  topology_json = jsonencode({
    data = [
      {
        type  = "MIRROR"
        disks = ["sda", "sdb"]
      }
    ]
  })

  encryption = false
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the pool (1-50 characters). Changing this attribute forces a new resource to be created.
* `topology_json` - (Optional) The pool topology as a raw JSON object. Must contain at least a `data` key with a list of vdev definitions (e.g. {"data":[{"type":"MIRROR","disks":["sda","sdb"]}]}). May also contain cache, log, spares, special, and dedup keys. Required on create; ignored after import since the API does not round-trip the original request form. Changing this attribute forces a new resource to be created.
* `encryption` - (Optional) Whether to create a ZFS-encrypted root dataset for this pool.
* `encryption_options_json` - (Optional) Optional encryption options as a raw JSON object (e.g. generate_key, algorithm, passphrase, key, pbkdf2iters). Changing this attribute forces a new resource to be created.
* `deduplication` - (Optional) Deduplication mode: ON, VERIFY, OFF, or unset. Valid values: `ON`, `VERIFY`, `OFF`.
* `checksum` - (Optional) Checksum algorithm: ON, OFF, FLETCHER2, FLETCHER4, SHA256, SHA512, SKEIN, EDONR, BLAKE3. Valid values: `ON`, `OFF`, `FLETCHER2`, `FLETCHER4`, `SHA256`, `SHA512`, `SKEIN`, `EDONR`, `BLAKE3`.
* `allow_duplicate_serials` - (Optional) Whether to allow disks with duplicate serial numbers in this pool.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_pool` resource.
* `guid` - The ZFS GUID of the pool.
* `path` - The filesystem mount path of the pool.
* `status` - The current status of the pool (ONLINE, DEGRADED, FAULTED, ...).
* `healthy` - Whether the pool is in a healthy state.

## Import

The `truenas_pool` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
# Pools are imported by their numeric ID
terraform import truenas_pool.example 1
```
