data "truenas_pool" "tank" {
  name = "tank"
}

resource "truenas_scrub_task" "example" {
  pool            = data.truenas_pool.tank.id
  threshold       = 35
  description     = "Weekly scrub"
  enabled         = true
  schedule_minute = "0"
  schedule_hour   = "0"
  schedule_dow    = "0" # Sunday
}
