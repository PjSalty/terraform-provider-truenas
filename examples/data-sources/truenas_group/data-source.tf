data "truenas_group" "wheel" {
  name = "wheel"
}

output "wheel_gid" {
  value = data.truenas_group.wheel.gid
}
