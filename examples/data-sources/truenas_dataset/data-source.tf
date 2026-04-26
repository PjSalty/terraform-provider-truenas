data "truenas_dataset" "tank" {
  id = "tank"
}

output "dataset_used" {
  value = data.truenas_dataset.tank.used
}
