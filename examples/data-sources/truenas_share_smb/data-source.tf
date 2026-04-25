data "truenas_share_smb" "example" {
  id = 1
}

output "smb_name" {
  value = data.truenas_share_smb.example.name
}
