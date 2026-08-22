resource "truenas_smb_config" "this" {
  netbiosname      = "truenas"
  workgroup        = "WORKGROUP"
  description      = "TrueNAS SCALE SMB server"
  minimum_protocol = "SMB2"
  unixcharset      = "UTF-8"
  enable_smb1      = false
  aapl_extensions  = false
  guest            = "nobody"
  filemask         = "0775"
  dirmask          = "0775"
}
