# Push tank/data to another system after each run of periodic snapshot task 3,
# at most once every two hours. ssh_credentials is the ID of an
# SSH_CREDENTIALS keychain credential.
resource "truenas_replication" "example" {
  name             = "tank-offsite"
  direction        = "PUSH"
  transport        = "SSH"
  ssh_credentials  = 12
  source_datasets  = ["tank/data"]
  target_dataset   = "backup/tank"
  recursive        = true
  enabled          = true
  auto             = true
  retention_policy = "NONE"

  periodic_snapshot_tasks = [3]
  hold_pending_snapshots  = true
  restrict_schedule       = [{ hour = "*/2" }]
}

# Pull it back every six hours on the task's own schedule.
resource "truenas_replication" "scheduled" {
  name             = "offsite-pull"
  direction        = "PULL"
  transport        = "SSH"
  ssh_credentials  = 12
  source_datasets  = ["backup/tank"]
  target_dataset   = "tank/restore"
  auto             = true
  retention_policy = "NONE"
  naming_schema    = ["auto-%Y-%m-%d_%H-%M"]

  schedule = [{ hour = "*/6" }]
}
