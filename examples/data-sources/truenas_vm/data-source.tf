data "truenas_vm" "example" {
  id = 1
}

output "vm_state" {
  value = data.truenas_vm.example.status
}
