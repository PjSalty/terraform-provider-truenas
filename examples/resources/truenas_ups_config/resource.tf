resource "truenas_ups_config" "this" {
  identifier     = "ups"
  mode           = "MASTER"
  driver         = "usbhid-ups"
  port           = "auto"
  monuser        = "upsmon"
  monpwd         = "changeme"
  shutdown_mode  = "BATT"
  shutdown_timer = 30
  rmonitor       = false
  powerdown      = false
}
