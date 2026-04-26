resource "truenas_replication" "example" {
  name             = "tank-offsite"
  direction        = "PUSH"
  transport        = "SSH"
  source_datasets  = ["tank/data"]
  target_dataset   = "backup/tank"
  recursive        = true
  enabled          = true
  auto             = true
  retention_policy = "NONE"
}
