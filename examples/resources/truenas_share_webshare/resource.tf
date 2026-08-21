# WebShare is the browser-based share protocol TrueNAS added in 26.0.
# This resource cannot be used against 25.10 or older, which have no
# sharing.webshare namespace.

resource "truenas_dataset" "docs" {
  pool = "tank"
  name = "docs"
}

resource "truenas_share_webshare" "docs" {
  name = "docs"
  path = truenas_dataset.docs.mount_point

  enabled = true

  # Set on exactly one share to make it the home base, under which
  # per-user home shares are created.
  is_home_base = false
}
