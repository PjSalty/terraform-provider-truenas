data "truenas_iscsi_target" "example" {
  id = 1
}

output "iscsi_target_name" {
  value = data.truenas_iscsi_target.example.name
}
