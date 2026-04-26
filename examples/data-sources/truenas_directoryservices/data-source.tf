data "truenas_directoryservices" "current" {}

output "ds_type" {
  value = data.truenas_directoryservices.current.service_type
}
