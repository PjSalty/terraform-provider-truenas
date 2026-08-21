---
page_title: "truenas_share_webshare Resource - terraform-provider-truenas"
subcategory: "Sharing"
description: |-
  Manages a WebShare share on TrueNAS SCALE. WebShare is the browser-based share protocol introduced in TrueNAS 26.0.
---

# truenas_share_webshare (Resource)

Manages a WebShare share on TrueNAS SCALE.

~> **Requires TrueNAS 26.0 or newer.** WebShare was introduced in 26.0 and the
`sharing.webshare` API namespace does not exist on 25.10 or earlier. Against an
older server every operation fails with a diagnostic naming the required
version, rather than an opaque method-not-found.

## Example Usage

```hcl
resource "truenas_dataset" "docs" {
  pool = "tank"
  name = "docs"
}

resource "truenas_share_webshare" "docs" {
  name    = "docs"
  path    = truenas_dataset.docs.mount_point
  enabled = true
}
```

## Argument Reference

* `name` - (Required) The share name.
* `path` - (Required) Filesystem path exposed by the share.
* `enabled` - (Optional) Whether the share is available. Default: `true`,
  matching the server-side default.
* `is_home_base` - (Optional) Whether this share is the home base, under which
  per-user home shares are created. Default: `false`.

## Attribute Reference

These are derived by TrueNAS and cannot be set. They are excluded from the
upstream create model, so a config that tried to set them would be expressing
something the server ignores.

* `id` - The WebShare share ID.
* `dataset` - Dataset backing the share path.
* `relative_path` - Path relative to the backing dataset.
* `locked` - Whether the share's dataset is locked by encryption.

## Import

```shell
terraform import truenas_share_webshare.docs 1
```

The ID is the numeric WebShare share ID.
