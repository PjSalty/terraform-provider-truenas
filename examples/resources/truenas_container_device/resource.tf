# Devices attached to an LXC container. Requires TrueNAS 26.0 or newer.
#
# Exactly one of filesystem, gpu, nic or usb per resource: a device is one
# kind of thing. Use a separate resource for each device.

resource "truenas_container" "web" {
  name = "web"
  image = {
    name    = "alpine:3.21:amd64:default"
    version = "20260820_23:08"
  }
}

# Bind-mount a host path. The source must live under /mnt, so it is on a
# pool rather than the boot device.
resource "truenas_container_device" "media" {
  container = truenas_container.web.id

  filesystem = {
    source = "/mnt/tank/media"
    target = "/srv/media"
  }
}

# Attach to a bridge. Valid values for nic_attach come from
# container.device.nic_attach_choices; leave it empty to create the
# interface unattached, and leave mac unset to have TrueNAS generate one.
resource "truenas_container_device" "lan" {
  container = truenas_container.web.id

  nic = {
    type       = "VIRTIO"
    nic_attach = "truenasbr0"
  }
}

# Pass a USB device through, identified by its vendor and product IDs.
resource "truenas_container_device" "dongle" {
  container = truenas_container.web.id

  usb = {
    vendor_id  = "0x1d6b"
    product_id = "0x0002"
  }
}
