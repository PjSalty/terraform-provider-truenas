resource "truenas_cronjob" "example" {
  command     = "/usr/bin/zpool scrub tank"
  description = "Weekly tank scrub"
  user        = "root"
  enabled     = true
  stdout      = true
  stderr      = true

  # 3am every Sunday
  schedule_minute = "0"
  schedule_hour   = "3"
  schedule_dow    = "0"
}
