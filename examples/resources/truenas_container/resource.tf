# LXC container. Requires TrueNAS 26.0 or newer; the container API
# namespace does not exist on 25.10 or earlier.
#
# Devices (NIC, USB, filesystem, GPU) are attached with
# truenas_container_device, not from here.

resource "truenas_container" "web" {
  name = "web"

  # Name and version both come from container.image.query_registry.
  image = {
    name    = "alpine:3.21:amd64:default"
    version = "20260820_23:08"
  }

  description = "Front-end container"
  autostart   = true

  # Pool for the container's root filesystem. Leave unset to use the
  # preferred pool from truenas_lxc_config.
  pool = "tank"

  # Environment for the init process.
  initenv = {
    TZ = "UTC"
  }

  # ISOLATED offsets this container's host UIDs away from every other
  # container's. Omit slice to have TrueNAS pick an unused one.
  #
  # Leaving the idmap block out entirely gives the TrueNAS default
  # mapping, which is also safe. What you cannot express here is an
  # UNMAPPED container, where container root is host root; that is
  # deliberate.
  idmap = {
    type = "ISOLATED"
  }
}

output "container_dataset" {
  value = truenas_container.web.dataset
}

output "container_state" {
  value = truenas_container.web.status.state
}
