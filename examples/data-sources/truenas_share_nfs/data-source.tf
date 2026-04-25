data "truenas_share_nfs" "example" {
  id = 1
}

output "nfs_path" {
  value = data.truenas_share_nfs.example.path
}
