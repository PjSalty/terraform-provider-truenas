resource "truenas_dataset" "acl_data" {
  pool = "tank"
  name = "acl_data"
}

resource "truenas_filesystem_acl" "example" {
  path    = "/mnt/tank/acl_data"
  acltype = "NFS4"

  dacl = [
    {
      tag          = "owner@"
      type         = "ALLOW"
      perm_read    = true
      perm_write   = true
      perm_execute = true
      default      = false
    },
    {
      tag          = "group@"
      type         = "ALLOW"
      perm_read    = true
      perm_write   = true
      perm_execute = true
      default      = false
    },
    {
      tag          = "everyone@"
      type         = "ALLOW"
      perm_read    = true
      perm_write   = false
      perm_execute = true
      default      = false
    },
  ]
}
