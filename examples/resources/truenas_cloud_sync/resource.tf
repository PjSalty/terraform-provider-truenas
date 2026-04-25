resource "truenas_cloud_sync" "example" {
  description   = "Offsite backup to S3"
  path          = "/mnt/tank/backup"
  credentials   = 1 # ID of a cloud credential keychain entry
  direction     = "PUSH"
  transfer_mode = "SYNC"
  enabled       = true
  attributes_json = jsonencode({
    bucket = "example-backup"
    folder = "truenas"
  })
  schedule {
    minute = "0"
    hour   = "4"
  }
}
