resource "truenas_static_route" "example" {
  destination = "10.99.0.0/16"
  gateway     = "10.0.0.1"
  description = "Route to backup LAN"
}
