resource "truenas_dns_nameserver" "primary" {
  nameserver1 = "1.1.1.1"
  nameserver2 = "9.9.9.9"
  nameserver3 = "8.8.8.8"
}
