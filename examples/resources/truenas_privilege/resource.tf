resource "truenas_privilege" "example" {
  name         = "zfs-operators"
  local_groups = [] # filled in with numeric group IDs
  ds_groups    = []
  roles        = ["SHARING_NFS_WRITE", "SHARING_SMB_WRITE"]
  web_shell    = false
}
