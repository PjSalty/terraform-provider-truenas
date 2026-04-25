data "truenas_snapshot_task" "example" {
  id = 1
}

output "snapshot_task_dataset" {
  value = data.truenas_snapshot_task.example.dataset
}
