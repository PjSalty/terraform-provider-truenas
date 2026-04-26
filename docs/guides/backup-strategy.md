---
page_title: "Backup Strategy - TrueNAS Provider"
subcategory: "Guides"
description: |-
  Automating backups with TrueNAS snapshots and replication using Terraform.
---

# Backup Strategy

This guide covers designing and implementing a complete backup strategy using TrueNAS snapshot tasks and replication.

## Snapshot-Based Backup Tiers

A well-designed backup strategy implements multiple retention tiers to balance storage usage against recovery point objectives (RPO):

| Tier | Frequency | Retention | Purpose |
|---|---|---|---|
| Hourly | Every hour | 24 hours | Fine-grained recovery within the day |
| Daily | Every night at 2am | 30 days | Day-level recovery for the past month |
| Weekly | Every Sunday at 3am | 12 weeks | Week-level recovery for the past quarter |
| Monthly | 1st of month at 4am | 12 months | Month-level recovery for the past year |

## Complete Multi-Tier Snapshot Configuration

```terraform
locals {
  protected_dataset = "tank/important-data"
}

# Hourly snapshots — keep for 24 hours
resource "truenas_snapshot_task" "hourly" {
  dataset = local.protected_dataset
  recursive = true

  lifetime_value = 24
  lifetime_unit  = "HOUR"
  naming_schema  = "hourly-%Y-%m-%d_%H-%M"

  schedule_minute = "0"
  schedule_hour   = "*"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"
}

# Daily snapshots — keep for 30 days
resource "truenas_snapshot_task" "daily" {
  dataset = local.protected_dataset
  recursive = true

  lifetime_value = 30
  lifetime_unit  = "DAY"
  naming_schema  = "daily-%Y-%m-%d"

  schedule_minute = "0"
  schedule_hour   = "2"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"
}

# Weekly snapshots — keep for 12 weeks
resource "truenas_snapshot_task" "weekly" {
  dataset = local.protected_dataset
  recursive = true

  lifetime_value = 12
  lifetime_unit  = "WEEK"
  naming_schema  = "weekly-%Y-W%V"

  schedule_minute = "0"
  schedule_hour   = "3"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "0"  # Sunday
}

# Monthly snapshots — keep for 12 months
resource "truenas_snapshot_task" "monthly" {
  dataset = local.protected_dataset
  recursive = true

  lifetime_value = 12
  lifetime_unit  = "MONTH"
  naming_schema  = "monthly-%Y-%m"

  schedule_minute = "0"
  schedule_hour   = "4"
  schedule_dom    = "1"
  schedule_month  = "*"
  schedule_dow    = "*"
}
```

## Local Pool-to-Pool Replication

Replicate from a primary pool to a backup pool on the same server. This protects against dataset-level accidents but not hardware failure.

```terraform
# Backup pool dataset to receive replicated data
resource "truenas_dataset" "backup_target" {
  pool     = "backup"
  name     = "important-data-replica"
  readonly = "ON"  # Prevent accidental writes to backup
}

resource "truenas_replication" "local_backup" {
  name            = "tank-to-backup-pool"
  direction       = "PUSH"
  transport       = "LOCAL"
  source_datasets = ["tank/important-data"]
  target_dataset  = "backup/important-data-replica"

  recursive        = true
  auto             = true
  retention_policy = "SOURCE"
}
```

## Offsite SSH Replication

Replicate to a remote TrueNAS server for geographic redundancy. Requires an SSH connection configured in TrueNAS (**Credentials → Backup Credentials → SSH Connections**).

```terraform
variable "ssh_credentials_id" {
  description = "ID of the SSH connection in TrueNAS Backup Credentials"
  type        = number
}

resource "truenas_replication" "offsite" {
  name            = "offsite-replication"
  direction       = "PUSH"
  transport       = "SSH"
  source_datasets = [
    "tank/important-data",
    "tank/databases",
    "tank/media",
  ]
  target_dataset  = "offsite-tank/truenas-backup"

  ssh_credentials  = var.ssh_credentials_id
  recursive        = true
  auto             = true
  retention_policy = "CUSTOM"
  lifetime_value   = 90
  lifetime_unit    = "DAY"
}
```

## Monitoring with Alert Services

Configure alerts to notify you of backup failures:

```terraform
resource "truenas_alert_service" "backup_alerts" {
  name    = "Backup Failure Alerts"
  type    = "Slack"
  enabled = true
  level   = "ERROR"

  settings_json = jsonencode({
    url = var.slack_webhook_url
  })
}
```

## Automated Backup Verification Cron Job

```terraform
resource "truenas_cronjob" "verify_backups" {
  user        = "root"
  command     = "/usr/local/bin/verify-backup-snapshots.sh"
  description = "Daily backup snapshot verification"

  schedule_minute = "30"
  schedule_hour   = "6"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"

  stdout = false
  stderr = true  # Email errors to root
}
```

## Data Source: Verify Pool Health Before Backup

```terraform
data "truenas_pool" "tank" {
  name = "tank"
}

data "truenas_pool" "backup" {
  name = "backup"
}

# Use a check to ensure pools are healthy before proceeding
check "pools_healthy" {
  assert {
    condition     = data.truenas_pool.tank.healthy && data.truenas_pool.backup.healthy
    error_message = "One or more pools are unhealthy — backup configuration may be at risk."
  }
}
```

## RPO and RTO Reference

| Scenario | Recovery Point | Recovery Time | Method |
|---|---|---|---|
| Accidental file deletion | Last hourly snapshot (up to 1h) | Minutes | ZFS snapshot rollback |
| Dataset corruption | Last daily snapshot (up to 24h) | Minutes to hours | ZFS snapshot clone |
| Pool failure | Last replication (up to 24h) | Hours | Restore from replica pool |
| Site failure | Last offsite replication | Hours to days | Restore from offsite |
