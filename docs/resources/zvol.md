---
page_title: "truenas_zvol Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages a ZFS volume (zvol) on TrueNAS SCALE.
---

# truenas_zvol (Resource)

Manages a ZFS volume (zvol) on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_zvol" "example" {
  pool         = "tank"
  name         = "vols/example"
  volsize      = 10737418240 # 10 GiB
  volblocksize = "16K"
  compression  = "LZ4"
  comments     = "Example zvol for iSCSI"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the zvol (without pool prefix). Changing this attribute forces a new resource to be created.
* `pool` - (Required) The pool to create the zvol in. Changing this attribute forces a new resource to be created.
* `volsize` - (Required) The size of the zvol in bytes.
* `volblocksize` - (Optional) The block size of the zvol (e.g., 4K, 8K, 16K, 32K, 64K, 128K). Valid values: `512`, `1K`, `2K`, `4K`, `8K`, `16K`, `32K`, `64K`, `128K`. Default: `16K`. Changing this attribute forces a new resource to be created.
* `deduplication` - (Optional) Deduplication setting (ON, OFF, VERIFY). Valid values: `ON`, `OFF`, `VERIFY`.
* `compression` - (Optional) Compression algorithm. A zvol takes the same set as a dataset, since it is a dataset of type VOLUME. Valid values: `OFF`, `ON`, `INHERIT`, `LZ4`, `LZJB`, `ZLE`, `GZIP`, `GZIP-1`, `GZIP-9`, `ZSTD`, `ZSTD-FAST`, `ZSTD-1`, `ZSTD-2`, `ZSTD-3`, `ZSTD-4`, `ZSTD-5`, `ZSTD-6`, `ZSTD-7`, `ZSTD-8`, `ZSTD-9`, `ZSTD-10`, `ZSTD-11`, `ZSTD-12`, `ZSTD-13`, `ZSTD-14`, `ZSTD-15`, `ZSTD-16`, `ZSTD-17`, `ZSTD-18`, `ZSTD-19`, `ZSTD-FAST-1`, `ZSTD-FAST-2`, `ZSTD-FAST-3`, `ZSTD-FAST-4`, `ZSTD-FAST-5`, `ZSTD-FAST-6`, `ZSTD-FAST-7`, `ZSTD-FAST-8`, `ZSTD-FAST-9`, `ZSTD-FAST-10`, `ZSTD-FAST-20`, `ZSTD-FAST-30`, `ZSTD-FAST-40`, `ZSTD-FAST-50`, `ZSTD-FAST-60`, `ZSTD-FAST-70`, `ZSTD-FAST-80`, `ZSTD-FAST-90`, `ZSTD-FAST-100`, `ZSTD-FAST-500`, `ZSTD-FAST-1000`.
* `comments` - (Optional) User-provided comments for the zvol.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_zvol` resource.

## Import

The `truenas_zvol` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
# Zvols are imported by their full ZFS path
terraform import truenas_zvol.example tank/vols/example
```
