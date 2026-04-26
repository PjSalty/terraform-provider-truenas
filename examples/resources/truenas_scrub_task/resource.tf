data "truenas_pool" "tank" {
  name = "tank"
}

resource "truenas_scrub_task" "example" {
  pool        = data.truenas_pool.tank.id
  threshold   = 35
  description = "Weekly scrub"
  enabled     = true
  schedule {
    minute = "0"
    hour   = "0"
    dow    = "0" # Sunday
  }
}
