resource "truenas_rsync_task" "example" {
  path            = "/mnt/tank/backup"
  user            = "root"
  mode            = "SSH"
  remotehost      = "backup.example.com"
  remotepath      = "/srv/backup"
  direction       = "PUSH"
  enabled         = true
  schedule_minute = "0"
  schedule_hour   = "2"
}
