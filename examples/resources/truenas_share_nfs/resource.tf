resource "truenas_dataset" "nfs_share" {
  pool = "tank"
  name = "nfs_share"
}

resource "truenas_share_nfs" "example" {
  path     = "/mnt/tank/nfs_share"
  comment  = "Example NFS export"
  enabled  = true
  networks = ["10.0.0.0/16"]

  maproot_user  = "root"
  maproot_group = "wheel"
}
