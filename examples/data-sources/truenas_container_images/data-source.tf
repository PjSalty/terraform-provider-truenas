# Image versions are datestamped and the registry keeps only the most
# recent few, so a hardcoded version stops resolving within days. Resolve
# it at plan time instead.
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

# Or pull a specific image's newest version straight out of the map.
output "alpine_latest" {
  value = data.truenas_container_images.alpine.latest_by_name["alpine:3.21:amd64:default"]
}
