resource "truenas_dataset" "snap_source" {
  pool = "tank"
  name = "snap_source"
}

resource "truenas_snapshot_task" "example" {
  dataset        = truenas_dataset.snap_source.id
  recursive      = true
  lifetime_value = 2
  lifetime_unit  = "WEEK"
  naming_schema  = "auto-%Y-%m-%d_%H-%M"
  schedule {
    minute = "0"
    hour   = "*/4"
  }
  enabled = true
}
