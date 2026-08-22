resource "truenas_dataset" "smb_share" {
  pool = "tank"
  name = "smb_share"
}

# Best for most clients. DEFAULT_SHARE accepts aapl_name_mangling,
# hostsallow and hostsdeny in options.
resource "truenas_share_smb" "example" {
  path      = "/mnt/tank/smb_share"
  name      = "example"
  comment   = "Example SMB share"
  purpose   = "DEFAULT_SHARE"
  enabled   = true
  browsable = true
  readonly  = false

  options = {
    hostsallow = ["192.168.1.0/24"]
  }
}

# EXTERNAL_SHARE is a DFS redirect rather than a share of local storage, so
# path is the literal "EXTERNAL" and remote_path carries the targets. It takes
# no other options.
resource "truenas_share_smb" "dfs_proxy" {
  path    = "EXTERNAL"
  name    = "archive"
  purpose = "EXTERNAL_SHARE"

  options = {
    remote_path = ["fileserver.example.com\\archive"]
  }
}

# A Time Machine target. TrueNAS creates a dataset per connecting user and
# snapshots it when a backup starts.
resource "truenas_share_smb" "timemachine" {
  path    = "/mnt/tank/smb_share"
  name    = "backups"
  purpose = "TIMEMACHINE_SHARE"

  options = {
    auto_dataset_creation = true
    auto_snapshot         = true
    dataset_naming_schema = "%D/%U"
  }
}
