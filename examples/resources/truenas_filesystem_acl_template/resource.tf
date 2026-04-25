resource "truenas_filesystem_acl_template" "example" {
  name    = "restricted-share"
  acltype = "NFS4"
  acl_json = jsonencode([
    {
      tag   = "owner@"
      type  = "ALLOW"
      perms = { BASIC = "FULL_CONTROL" }
      flags = { BASIC = "INHERIT" }
    }
  ])
}
