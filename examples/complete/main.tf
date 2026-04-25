terraform {
  required_providers {
    truenas = {
      source  = "registry.terraform.io/salty/truenas"
      version = "~> 0.1"
    }
  }
}

provider "truenas" {
  url     = var.truenas_url
  api_key = var.truenas_api_key
}

variable "truenas_url" {
  description = "TrueNAS SCALE URL"
  type        = string
}

variable "truenas_api_key" {
  description = "TrueNAS API key"
  type        = string
  sensitive   = true
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for alerts"
  type        = string
  sensitive   = true
  default     = ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Data sources: read existing infrastructure
# ─────────────────────────────────────────────────────────────────────────────

data "truenas_system_info" "this" {}

data "truenas_pool" "tank" {
  name = "tank"
}

# ─────────────────────────────────────────────────────────────────────────────
# Datasets
# ─────────────────────────────────────────────────────────────────────────────

resource "truenas_dataset" "apps" {
  pool        = data.truenas_pool.tank.name
  name        = "apps"
  compression = "LZ4"
  atime       = "OFF"
  comments    = "Parent dataset for application data"
}

resource "truenas_dataset" "media" {
  pool        = data.truenas_pool.tank.name
  name        = "media"
  compression = "OFF" # Media files are already compressed
  atime       = "OFF"
  share_type  = "SMB"
  comments    = "Media library"
}

resource "truenas_dataset" "k8s_nfs" {
  pool        = data.truenas_pool.tank.name
  name        = "k8s-nfs"
  compression = "LZ4"
  atime       = "OFF"
  share_type  = "GENERIC"
  comments    = "Kubernetes NFS persistent volumes"
}

# ─────────────────────────────────────────────────────────────────────────────
# Shares
# ─────────────────────────────────────────────────────────────────────────────

resource "truenas_share_nfs" "k8s" {
  path     = truenas_dataset.k8s_nfs.mount_point
  comment  = "Kubernetes NFS PVCs"
  networks = ["10.0.0.0/24"]

  maproot_user  = "root"
  maproot_group = "wheel"
}

resource "truenas_share_smb" "media" {
  path      = truenas_dataset.media.mount_point
  name      = "media"
  comment   = "Media library"
  browsable = true
  readonly  = true
}

# ─────────────────────────────────────────────────────────────────────────────
# iSCSI stack for databases
# ─────────────────────────────────────────────────────────────────────────────

resource "truenas_iscsi_portal" "main" {
  comment = "Primary iSCSI portal"

  listen {
    ip   = "10.0.0.10"
    port = 3260
  }
}

resource "truenas_iscsi_initiator" "k8s_nodes" {
  comment = "Kubernetes worker nodes"

  initiators = [
    "iqn.2025-01.com.example:k8s-worker-1",
    "iqn.2025-01.com.example:k8s-worker-2",
    "iqn.2025-01.com.example:k8s-worker-3",
  ]
}

resource "truenas_dataset" "postgres_zvol" {
  pool = data.truenas_pool.tank.name
  name = "postgres-data"
  type = "VOLUME"
}

resource "truenas_iscsi_extent" "postgres" {
  name    = "postgres-data"
  type    = "DISK"
  disk    = "zvol/${truenas_dataset.postgres_zvol.id}"
  comment = "PostgreSQL data volume"

  blocksize = 4096
  rpm       = "SSD"
}

resource "truenas_iscsi_target" "postgres" {
  name  = "iqn.2025-01.com.example:postgres-data"
  alias = "postgres-data"

  groups {
    portal    = truenas_iscsi_portal.main.tag
    initiator = tonumber(truenas_iscsi_initiator.k8s_nodes.id)
  }
}

# ─────────────────────────────────────────────────────────────────────────────
# Snapshot tasks
# ─────────────────────────────────────────────────────────────────────────────

resource "truenas_snapshot_task" "k8s_daily" {
  dataset = truenas_dataset.k8s_nfs.id

  recursive      = true
  lifetime_value = 30
  lifetime_unit  = "DAY"
  naming_schema  = "daily-%Y-%m-%d"

  schedule_minute = "0"
  schedule_hour   = "2"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"
}

resource "truenas_snapshot_task" "media_weekly" {
  dataset = truenas_dataset.media.id

  recursive      = false
  lifetime_value = 8
  lifetime_unit  = "WEEK"
  naming_schema  = "weekly-%Y-W%V"

  schedule_minute = "0"
  schedule_hour   = "3"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "0"
}

# ─────────────────────────────────────────────────────────────────────────────
# Replication
# ─────────────────────────────────────────────────────────────────────────────

resource "truenas_replication" "local_backup" {
  name            = "tank-to-backup"
  direction       = "PUSH"
  transport       = "LOCAL"
  source_datasets = [truenas_dataset.apps.id]
  target_dataset  = "backup/apps-replica"

  recursive        = true
  auto             = true
  retention_policy = "SOURCE"
}

# ─────────────────────────────────────────────────────────────────────────────
# Alert service
# ─────────────────────────────────────────────────────────────────────────────

resource "truenas_alert_service" "slack" {
  count = var.slack_webhook_url != "" ? 1 : 0

  name    = "Slack Alerts"
  type    = "Slack"
  enabled = true
  level   = "WARNING"

  settings_json = jsonencode({
    url = var.slack_webhook_url
  })
}

# ─────────────────────────────────────────────────────────────────────────────
# Cron job
# ─────────────────────────────────────────────────────────────────────────────

resource "truenas_cronjob" "scrub" {
  user        = "root"
  command     = "zpool scrub tank"
  description = "Monthly pool scrub"

  schedule_minute = "0"
  schedule_hour   = "2"
  schedule_dom    = "1"
  schedule_month  = "*"
  schedule_dow    = "*"
}

# ─────────────────────────────────────────────────────────────────────────────
# Outputs
# ─────────────────────────────────────────────────────────────────────────────

output "truenas_version" {
  value = data.truenas_system_info.this.version
}

output "nfs_mount_path" {
  value = truenas_dataset.k8s_nfs.mount_point
}

output "iscsi_portal_tag" {
  value = truenas_iscsi_portal.main.tag
}
