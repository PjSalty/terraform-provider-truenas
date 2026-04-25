resource "truenas_kmip_config" "this" {
  server                = "kmip.example.com"
  port                  = 5696
  certificate           = 1 # Certificate resource ID
  certificate_authority = 2 # CA resource ID
  manage_sed_disks      = false
  manage_zfs_keys       = false
  enabled               = false
}
