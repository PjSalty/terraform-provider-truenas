data "truenas_privilege" "admin" {
  id = 1
}

output "privilege_name" {
  value = data.truenas_privilege.admin.name
}
