resource "truenas_smb_config" "this" {
  netbiosname = "truenas"
  workgroup   = "WORKGROUP"
  description = "TrueNAS SCALE SMB server"
  minimum_protocol = "SMB2"
  unixcharset = "UTF-8"
  loglevel    = "MINIMUM"
  syslog      = false
  localmaster = true
  guest       = "nobody"
  filemask    = "0775"
  dirmask     = "0775"
  ntlmv1_auth = false
}
