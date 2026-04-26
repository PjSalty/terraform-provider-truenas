data "truenas_catalog" "official" {}

output "catalog_label" {
  value = data.truenas_catalog.official.label
}
