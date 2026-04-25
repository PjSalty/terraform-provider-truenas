resource "truenas_network_interface" "example" {
  type      = "PHYSICAL"
  name      = "eno1"
  dhcp      = false
  ipv6_auto = false

  aliases = [
    {
      type    = "INET"
      address = "10.0.0.20"
      netmask = 24
    }
  ]
  mtu = 1500
}
