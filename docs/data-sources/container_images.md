---
page_title: "truenas_container_images Data Source - terraform-provider-truenas"
subcategory: "Virtualization"
description: |-
  Lists the images available to create LXC containers from on TrueNAS SCALE. Requires TrueNAS 26.0 or newer.
---

# truenas_container_images (Data Source)

Lists the images an LXC container can be created from, with every version the
registry currently publishes for each.

~> **Requires TrueNAS 26.0 or newer.** The `container.image` API namespace does
not exist on 25.10 or earlier.

Image versions are datestamped and the registry keeps only the most recent few,
so a version hardcoded in a configuration stops resolving within days. Resolving
it here keeps `truenas_container` working without edits.

Reading this queries the upstream image registry, so it is slower than a local
read and fails if the server has no route to it.

## Example Usage

```terraform
data "truenas_container_images" "alpine" {
  name_prefix = "alpine:3.21:"
}

resource "truenas_container" "web" {
  name = "web"

  image = {
    name    = one(data.truenas_container_images.alpine.images).name
    version = one(data.truenas_container_images.alpine.images).latest_version
  }
}

output "alpine_latest" {
  value = data.truenas_container_images.alpine.latest_by_name["alpine:3.21:amd64:default"]
}
```

## Argument Reference

* `name_prefix` - (Optional) Only return images whose name starts with this, for
  example `alpine:`. Omit to return everything the registry publishes.

## Attribute Reference

* `id` - Always `container_images`.
* `images` - Matching images, sorted by name.
  * `name` - Image name, for example `alpine:3.21:amd64:default`.
  * `versions` - Published versions, oldest first, as the registry returns them.
  * `latest_version` - The newest published version, which is the one least
    likely to age out. Empty for an image with no published versions.
* `names` - Just the matching image names, sorted.
* `latest_by_name` - Newest version keyed by image name.

## Behavior notes

### Results are sorted

The registry's own ordering is not documented as stable. Sorting by name here
means the list does not churn between plans for no reason.

### An image with no versions is still listed

Such an image is reported with an empty `latest_version` rather than being
dropped, so a configuration referring to it fails on the empty value instead of
on a missing map key, which is the harder failure to read.
