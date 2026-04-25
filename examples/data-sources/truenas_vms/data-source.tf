data "truenas_vms" "all" {}

output "vm_names" {
  value = [for v in data.truenas_vms.all.vms : v.name]
}
