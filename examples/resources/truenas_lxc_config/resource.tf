# System-wide LXC container configuration. Requires TrueNAS 26.0 or newer;
# the lxc API namespace does not exist on 25.10 or earlier.
#
# This is a singleton: TrueNAS has exactly one LXC configuration.

resource "truenas_lxc_config" "this" {
  # Default pool for container and image datasets.
  preferred_pool = "tank"

  # Leave bridge empty to have TrueNAS manage one automatically, or name an
  # existing bridge. A named bridge is validated against lxc.bridge_choices
  # at apply time.
  bridge = ""

  # These are the TrueNAS defaults. They are deliberately clear of the
  # Docker address pools (172.16.0.0/12, fdd0::/48) and of the 10.x.x.0/24
  # ranges Incus generates, so containers do not collide with either.
  # Override only if they clash with your own networks.
  v4_network = "172.200.0.0/24"
  v6_network = "fd42:4c58:43ae::/64"
}
