resource "truenas_dataset" "smb_share" {
  pool = "tank"
  name = "smb_share"
}

resource "truenas_share_smb" "example" {
  path      = "/mnt/tank/smb_share"
  name      = "example"
  comment   = "Example SMB share"
  purpose   = "DEFAULT_SHARE"
  enabled   = true
  browsable = true
  home      = false
  read_only = false
  guestok   = false
}
