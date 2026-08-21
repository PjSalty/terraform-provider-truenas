# LXC containers are introduced in TrueNAS 26.0. Against 25.10 or older
# this data source errors: the lxc namespace does not exist there.
data "truenas_lxc_config" "current" {}

output "container_pool" {
  value = data.truenas_lxc_config.current.preferred_pool
}

# Empty means TrueNAS creates and manages the bridge itself.
output "container_bridge" {
  value = data.truenas_lxc_config.current.bridge
}
