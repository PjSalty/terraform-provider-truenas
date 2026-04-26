resource "truenas_ftp_config" "this" {
  port          = 21
  clients       = 32
  ipconnections = 5
  loginattempt  = 3
  timeout       = 600
  rootlogin     = false
  onlyanonymous = false
  onlylocal     = false
  defaultroot   = true
  ident         = false
  fxp           = false
  resume        = false
  ssltls_policy = "on"
}
