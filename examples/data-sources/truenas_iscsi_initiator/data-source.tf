data "truenas_iscsi_initiator" "example" {
  id = 1
}

output "iscsi_initiator_list" {
  value = data.truenas_iscsi_initiator.example.initiators
}
