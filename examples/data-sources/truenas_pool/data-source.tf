data "truenas_pool" "tank" {
  name = "tank"
}

output "pool_status" {
  value = data.truenas_pool.tank.status
}
