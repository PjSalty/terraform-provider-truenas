data "truenas_apps" "all" {}

output "app_names" {
  value = [for a in data.truenas_apps.all.apps : a.name]
}
