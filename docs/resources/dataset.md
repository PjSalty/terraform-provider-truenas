---
page_title: "truenas_dataset Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages a ZFS dataset on TrueNAS SCALE.
---

# truenas_dataset (Resource)

Manages a ZFS dataset on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_dataset" "example" {
  pool        = "tank"
  name        = "example"
  compression = "LZ4"
  atime       = "OFF"
  comments    = "Example dataset managed by Terraform"
}

# Child dataset under an existing parent
resource "truenas_dataset" "child" {
  pool           = "tank"
  parent_dataset = truenas_dataset.example.name
  name           = "child"
  compression    = "LZ4"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the dataset (without pool prefix). Changing this attribute forces a new resource to be created.
* `pool` - (Required) The pool to create the dataset in. Changing this attribute forces a new resource to be created.
* `parent_dataset` - (Optional) Optional parent dataset path relative to pool (e.g., 'parent/child'). The full dataset path will be pool/parent_dataset/name. Changing this attribute forces a new resource to be created.
* `type` - (Optional) The dataset type: FILESYSTEM or VOLUME. Valid values: `FILESYSTEM`, `VOLUME`. Default: `FILESYSTEM`. Changing this attribute forces a new resource to be created.
* `compression` - (Optional) Compression algorithm (OFF, LZ4, GZIP, ZSTD, ZLE, LZJB). Valid values: `OFF`, `ON`, `LZ4`, `GZIP`, `GZIP-1`, `GZIP-2`, `GZIP-3`, `GZIP-4`, `GZIP-5`, `GZIP-6`, `GZIP-7`, `GZIP-8`, `GZIP-9`, `ZSTD`, `ZSTD-FAST`, `ZLE`, `LZJB`, `INHERIT`.
* `atime` - (Optional) Access time update behavior (ON, OFF). Valid values: `ON`, `OFF`, `INHERIT`.
* `deduplication` - (Optional) Deduplication setting (ON, OFF, VERIFY). Valid values: `ON`, `OFF`, `VERIFY`, `INHERIT`.
* `quota` - (Optional) Dataset quota in bytes. 0 means no quota.
* `refquota` - (Optional) Dataset reference quota in bytes. 0 means no refquota.
* `comments` - (Optional) User-provided comments for the dataset.
* `sync` - (Optional) Sync write behavior (STANDARD, ALWAYS, DISABLED). Valid values: `STANDARD`, `ALWAYS`, `DISABLED`, `INHERIT`.
* `snapdir` - (Optional) Snapshot directory visibility (VISIBLE, HIDDEN). Valid values: `VISIBLE`, `HIDDEN`, `INHERIT`.
* `copies` - (Optional) Number of data copies (1, 2, or 3).
* `readonly` - (Optional) Read-only setting (ON, OFF). Valid values: `ON`, `OFF`, `INHERIT`.
* `record_size` - (Optional) Record size (e.g., 128K, 1M). Valid values: 512, 1K, 2K, 4K, 8K, 16K, 32K, 64K, 128K, 256K, 512K, 1M. Valid values: `512`, `1K`, `2K`, `4K`, `8K`, `16K`, `32K`, `64K`, `128K`, `256K`, `512K`, `1M`, `INHERIT`.
* `share_type` - (Optional) Share type preset (GENERIC, SMB). Valid values: `GENERIC`, `SMB`, `MULTIPROTOCOL`, `NFS`, `APPS`.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_dataset` resource.
* `mount_point` - The mount point of the dataset.

## Import

The `truenas_dataset` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
# Datasets are imported by their full ZFS path
terraform import truenas_dataset.example tank/example
```
