---
page_title: "Importing Existing Resources - TrueNAS Provider"
subcategory: "Guides"
description: |-
  How to import existing TrueNAS resources into Terraform state.
---

# Importing Existing Resources

If you have an existing TrueNAS configuration that was created manually (or by another tool), you can bring it under Terraform management using the `terraform import` command or the `import` block (Terraform 1.5+).

## Finding Resource IDs

Before importing, you need to know each resource's identifier. Most TrueNAS resources use numeric IDs assigned by the API, except datasets which use their full ZFS path.

| Resource | Import ID | Where to Find It |
|---|---|---|
| `truenas_dataset` | Full ZFS path (e.g., `tank/mydata`) | TrueNAS UI → Storage → Datasets |
| `truenas_share_nfs` | Numeric ID | TrueNAS UI → Sharing → NFS |
| `truenas_share_smb` | Numeric ID | TrueNAS UI → Sharing → SMB |
| `truenas_snapshot_task` | Numeric ID | TrueNAS UI → Data Protection → Periodic Snapshot Tasks |
| `truenas_replication` | Numeric ID | TrueNAS UI → Data Protection → Replication Tasks |
| `truenas_iscsi_portal` | Numeric ID | TrueNAS UI → Sharing → iSCSI → Portals |
| `truenas_iscsi_initiator` | Numeric ID | TrueNAS UI → Sharing → iSCSI → Initiators |
| `truenas_iscsi_extent` | Numeric ID | TrueNAS UI → Sharing → iSCSI → Extents |
| `truenas_iscsi_target` | Numeric ID | TrueNAS UI → Sharing → iSCSI → Targets |
| `truenas_cronjob` | Numeric ID | TrueNAS UI → System → Advanced → Cron Jobs |
| `truenas_alert_service` | Numeric ID | TrueNAS UI → System → Alert Settings |

You can also use the TrueNAS API directly to list resources and find their IDs:

```bash
# List all NFS shares with their IDs
curl -s -H "Authorization: Bearer $TRUENAS_API_KEY" \
  "$TRUENAS_URL/api/v2.0/sharing/nfs" | jq '.[] | {id, path}'

# List all datasets
curl -s -H "Authorization: Bearer $TRUENAS_API_KEY" \
  "$TRUENAS_URL/api/v2.0/pool/dataset" | jq '.[] | .id'

# List snapshot tasks
curl -s -H "Authorization: Bearer $TRUENAS_API_KEY" \
  "$TRUENAS_URL/api/v2.0/pool/snapshottask" | jq '.[] | {id, dataset}'
```

## Method 1: CLI Import (Terraform < 1.5)

First, write a resource block that matches the existing configuration:

```terraform
resource "truenas_dataset" "existing" {
  pool = "tank"
  name = "mydata"
}
```

Then import:

```bash
terraform import truenas_dataset.existing tank/mydata
```

After import, run `terraform plan` to see if the configuration matches the actual resource. If there are differences, update your configuration to match.

## Method 2: Import Blocks (Terraform 1.5+)

The `import` block is more declarative and can be committed to version control:

```terraform
import {
  to = truenas_dataset.existing
  id = "tank/mydata"
}

resource "truenas_dataset" "existing" {
  pool = "tank"
  name = "mydata"
}
```

Run `terraform plan` to preview the import, then `terraform apply` to complete it.

## Method 3: Generated Configuration (Terraform 1.5+)

For bulk imports, Terraform can generate the configuration for you:

```terraform
import {
  to = truenas_share_nfs.share_1
  id = "1"
}

import {
  to = truenas_share_nfs.share_2
  id = "2"
}
```

Run with `-generate-config-out` to produce configuration:

```bash
terraform plan -generate-config-out=generated.tf
```

Review `generated.tf`, clean it up, and move it into your main configuration.

## Importing a Complete Setup

Here is a realistic workflow for importing an existing TrueNAS configuration:

### Step 1: Audit Existing Resources

```bash
# List all datasets under 'tank'
curl -s -H "Authorization: Bearer $TRUENAS_API_KEY" \
  "$TRUENAS_URL/api/v2.0/pool/dataset" | \
  jq '.[] | select(.id | startswith("tank")) | .id'

# List NFS shares
curl -s -H "Authorization: Bearer $TRUENAS_API_KEY" \
  "$TRUENAS_URL/api/v2.0/sharing/nfs" | jq '.[] | {id, path, comment}'

# List snapshot tasks
curl -s -H "Authorization: Bearer $TRUENAS_API_KEY" \
  "$TRUENAS_URL/api/v2.0/pool/snapshottask" | jq '.[] | {id, dataset, enabled}'
```

### Step 2: Write Placeholder Resources

```terraform
resource "truenas_dataset" "media" {
  pool = "tank"
  name = "media"
}

resource "truenas_share_nfs" "media" {
  path = "/mnt/tank/media"
}

resource "truenas_snapshot_task" "media_daily" {
  dataset = "tank/media"
}
```

### Step 3: Import Each Resource

```bash
terraform import truenas_dataset.media tank/media
terraform import truenas_share_nfs.media 1
terraform import truenas_snapshot_task.media_daily 3
```

### Step 4: Reconcile Configuration

```bash
terraform plan
```

Review the diff. Any attributes shown as changing need to be added to your configuration to match the actual state. Common attributes that need explicit values after import:

- `compression`, `atime`, `sync`, `snapdir` — ZFS properties inherited from pool defaults
- `schedule_*` fields — cron schedule fields
- `lifetime_value`, `lifetime_unit` — retention settings

### Step 5: Iterate Until Clean Plan

Continue updating your configuration until `terraform plan` shows **No changes**.

```bash
terraform plan
# Apply: no changes. Your infrastructure matches the configuration.
```

## Tips for a Smooth Import

**Start with datasets first.** Other resources depend on dataset paths, so import those before shares or snapshot tasks.

**Import one resource at a time.** After each import, run `terraform plan` to catch discrepancies before moving on.

**Use `terraform state show` to inspect imported state:**

```bash
terraform state show truenas_dataset.media
```

**Don't modify resources before importing.** Wait until the resource is fully imported and your configuration matches before making changes.

**Handle computed attributes carefully.** Attributes like `id` and `mount_point` are computed by TrueNAS — don't include them in your configuration.
