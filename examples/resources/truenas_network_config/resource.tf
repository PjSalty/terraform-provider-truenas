resource "truenas_network_config" "this" {
  hostname        = "truenas"
  domain          = "example.com"
  nameserver1     = "1.1.1.1"
  nameserver2     = "9.9.9.9"
  ipv4gateway     = "10.0.0.1"
  httpproxy       = ""
  netwait_enabled = false
}
