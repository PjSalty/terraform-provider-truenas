data "truenas_disk" "sda" {
  name = "sda"
}

output "disk_model" {
  value = data.truenas_disk.sda.model
}
