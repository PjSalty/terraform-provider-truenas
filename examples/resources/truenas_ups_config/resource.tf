resource "truenas_ups_config" "this" {
  identifier    = "ups"
  description   = "Rack 1 UPS"
  mode          = "MASTER"
  driver        = "usbhid-ups"
  port          = "auto"
  shutdown      = "BATT"
  shutdowntimer = 30
}
