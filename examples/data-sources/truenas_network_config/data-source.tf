data "truenas_network_config" "current" {}

output "hostname" {
  value = data.truenas_network_config.current.hostname
}
