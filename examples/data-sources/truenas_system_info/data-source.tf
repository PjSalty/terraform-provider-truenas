data "truenas_system_info" "current" {}

output "version" {
  value = data.truenas_system_info.current.version
}
