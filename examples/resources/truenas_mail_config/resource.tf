resource "truenas_mail_config" "this" {
  fromemail      = "truenas@example.com"
  fromname       = "TrueNAS"
  outgoingserver = "smtp.example.com"
  port           = 587
  security       = "TLS"
  smtp           = true
  user           = "truenas"
  pass           = "smtp-password"
}
