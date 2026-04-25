data "truenas_pools" "all" {}

output "pool_names" {
  value = [for p in data.truenas_pools.all.pools : p.name]
}
