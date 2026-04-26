data "truenas_systemdataset" "current" {}

output "system_pool" {
  value = data.truenas_systemdataset.current.pool
}
