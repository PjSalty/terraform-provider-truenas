resource "truenas_ftp_config" "this" {
  port          = 21
  clients       = 32
  ipconnections = 5
  loginattempt  = 3
  timeout       = 600
  onlyanonymous = false
  onlylocal     = false
  defaultroot   = true
  fxp           = false
  resume        = false
  tls           = false
  filemask      = "0077"
  dirmask       = "0077"
}
