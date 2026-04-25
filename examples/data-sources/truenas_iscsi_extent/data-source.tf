data "truenas_iscsi_extent" "example" {
  id = 1
}

output "iscsi_extent_path" {
  value = data.truenas_iscsi_extent.example.path
}
