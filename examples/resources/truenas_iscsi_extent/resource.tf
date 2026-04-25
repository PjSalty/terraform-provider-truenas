resource "truenas_zvol" "iscsi_lun" {
  pool    = "tank"
  name    = "vols/iscsi-lun1"
  volsize = 10737418240 # 10 GiB
  sparse  = true
}

resource "truenas_iscsi_extent" "example" {
  name      = "iscsi-lun1"
  type      = "DISK"
  disk      = "zvol/tank/vols/iscsi-lun1"
  enabled   = true
  blocksize = 512
  rpm       = "SSD"
  comment   = "Example iSCSI LUN"
}
