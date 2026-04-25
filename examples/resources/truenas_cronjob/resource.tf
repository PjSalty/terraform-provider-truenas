resource "truenas_cronjob" "example" {
  command     = "/usr/bin/zpool scrub tank"
  description = "Weekly tank scrub"
  user        = "root"
  enabled     = true
  stdout      = true
  stderr      = true

  schedule {
    minute = "0"
    hour   = "3"
    dow    = "0"
  }
}
