---
page_title: "truenas_directory Resource - terraform-provider-truenas"
subcategory: "Storage"
description: |-
  Manages a directory on a TrueNAS SCALE filesystem path.
---

# truenas_directory (Resource)

Manages a directory on a TrueNAS SCALE filesystem path.

The directory is created via the TrueNAS JSON-RPC `filesystem.mkdir` endpoint
and its ownership and permissions are applied via `filesystem.setperm`. The
resource is keyed by its absolute path, so changing `path` forces a
replacement.

~> **Note** TrueNAS exposes no directory-removal API. On destroy the directory
is removed from Terraform state but left on disk; remove it manually if the
data is no longer needed.

-> **Note** `mode`, `uid`, and `gid` in state reflect the values the provider
applied. TrueNAS can serve a stale `filesystem.stat` right after `setperm`, so
the post-apply read only fills values the plan did not set. The regular
refresh read stays stat-authoritative for drift detection.

## Example Usage

### Basic

```terraform
resource "truenas_dataset" "media" {
  pool = "tank"
  name = "media"
}

resource "truenas_directory" "downloaded_music" {
  path           = "${truenas_dataset.media.mount_point}/downloaded/music"
  mode           = "755"
  create_parents = true
  uid            = 1000
  gid            = 1000
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The absolute filesystem path of the directory (e.g., /mnt/tank/media). Must be under `/mnt/`. Changing this forces a new resource.
* `mode` - (Optional) The octal permission mode for the directory, permission bits only (e.g., 755 or 0755). Default: `755`. The TrueNAS filesystem API rejects modes above 777, so setuid/setgid/sticky bits cannot be set through this resource.
* `create_parents` - (Optional) When `true`, create any missing parent directories before the leaf (like `mkdir -p`). Default: `false`.
* `uid` - (Optional) The owner UID. Applied via `setperm` when set or changed.
* `gid` - (Optional) The owner GID. Applied via `setperm` when set or changed.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_directory` resource (same as `path`).

## Import

The `truenas_directory` resource can be imported using its absolute path:

```shell
#!/usr/bin/env bash
terraform import truenas_directory.example /mnt/tank/media/downloaded/music
```

Imports default `create_parents` to `false`. The attribute only controls
mkdir -p behavior at create time, so it has no effect on an existing
directory; set it in config if a future replacement should create parents.
