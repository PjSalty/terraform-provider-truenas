resource "truenas_service" "ssh" {
  service = "ssh"
  enable  = true
  state   = "RUNNING"
}
