resource "truenas_rsync_task" "example" {
  path         = "/mnt/tank/backup"
  user         = "root"
  mode         = "SSH"
  remotehost   = "backup.example.com"
  remotepath   = "/srv/backup"
  direction    = "PUSH"
  enabled      = true
  recursive    = true
  times        = true
  compress     = true
  archive      = true
  delete       = false
  preserveperm = true
  schedule {
    minute = "0"
    hour   = "2"
  }
}
