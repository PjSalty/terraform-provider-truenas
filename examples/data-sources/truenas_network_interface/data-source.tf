data "truenas_network_interface" "eno1" {
  id = "eno1"
}

output "interface_state" {
  value = data.truenas_network_interface.eno1.state
}
