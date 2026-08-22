resource "truenas_cloud_backup" "example" {
  path        = "/mnt/tank/backup"
  credentials = 1 # numeric ID of a cloud credential
  password    = "restic-repo-password"
  keep_last   = 30
  enabled     = true
  attributes_json = jsonencode({
    bucket = "example-restic"
    folder = "truenas"
  })
  schedule_minute = "0"
  schedule_hour   = "5"
}
