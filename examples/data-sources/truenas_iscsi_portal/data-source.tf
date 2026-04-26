data "truenas_iscsi_portal" "example" {
  id = 1
}

output "iscsi_portal_tag" {
  value = data.truenas_iscsi_portal.example.tag
}
