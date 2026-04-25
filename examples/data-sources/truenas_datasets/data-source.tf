data "truenas_datasets" "all" {}

output "dataset_ids" {
  value = [for d in data.truenas_datasets.all.datasets : d.id]
}
