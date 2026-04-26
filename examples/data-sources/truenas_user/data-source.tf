data "truenas_user" "root" {
  username = "root"
}

output "root_uid" {
  value = data.truenas_user.root.uid
}
