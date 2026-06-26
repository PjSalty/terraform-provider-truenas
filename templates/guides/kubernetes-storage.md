---
page_title: "Kubernetes Storage - TrueNAS Provider"
subcategory: "Guides"
description: |-
  Using TrueNAS SCALE as persistent storage for Kubernetes clusters.
---

# Kubernetes Storage

This guide covers using TrueNAS SCALE as the backing storage for Kubernetes persistent volumes via NFS and iSCSI.

## Architecture Overview

TrueNAS SCALE integrates with Kubernetes through two primary protocols:

| Protocol | Use Case | Kubernetes Driver |
|---|---|---|
| NFS | ReadWriteMany (RWX) volumes, shared storage | `csi.truenas.io` (truenas-csi) |
| iSCSI | ReadWriteOnce (RWO) volumes, databases | `csi.truenas.io` (truenas-csi) |

## NFS Storage for Kubernetes (truenas-csi)

The recommended approach is to use [truenas-csi](https://github.com/truenas/truenas-csi), the official iX CSI driver, with TrueNAS as the backend. Terraform provisions the parent datasets; truenas-csi dynamically creates a per-PVC dataset and share for each volume. It talks the same JSON-RPC WebSocket API (`/api/current`) this provider uses, so it keeps working on TrueNAS 26, where the legacy REST API is removed. The older [democratic-csi](https://github.com/democratic-csi/democratic-csi) driver still works on 25.10 and earlier but is REST-only, so plan to move off it before upgrading to 26.

### Provision the Parent Dataset

```terraform
resource "truenas_dataset" "k8s_nfs" {
  pool        = "tank"
  name        = "k8s-nfs"
  compression = "LZ4"
  atime       = "OFF"
  share_type  = "GENERIC"
  comments    = "Parent dataset for Kubernetes NFS PVCs via truenas-csi"
}

resource "truenas_share_nfs" "k8s_nfs" {
  path     = truenas_dataset.k8s_nfs.mount_point
  comment  = "Kubernetes NFS PVC storage (truenas-csi)"
  networks = ["10.0.0.0/24"]  # Kubernetes pod/node CIDR

  maproot_user  = "root"
  maproot_group = "wheel"

  enabled = true
}
```

### Snapshot Tasks for PVC Protection

```terraform
resource "truenas_snapshot_task" "k8s_nfs_daily" {
  dataset = truenas_dataset.k8s_nfs.id

  recursive      = true
  lifetime_value = 7
  lifetime_unit  = "DAY"

  schedule_minute = "0"
  schedule_hour   = "1"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"

  naming_schema = "k8s-auto-%Y-%m-%d_%H-%M"
}
```

## iSCSI Storage for Databases

For databases and stateful workloads requiring block storage (ReadWriteOnce), iSCSI provides better performance and data consistency guarantees than NFS.

### Complete iSCSI Stack

```terraform
# 1. Portal: network endpoint
resource "truenas_iscsi_portal" "k8s" {
  comment = "Kubernetes iSCSI portal"

  listen {
    ip   = "10.0.0.10"  # Storage network IP
    port = 3260
  }
}

# 2. Initiator group: restrict to Kubernetes nodes
resource "truenas_iscsi_initiator" "k8s_nodes" {
  comment = "Kubernetes worker nodes"

  initiators = [
    "iqn.2025-01.com.example:k8s-worker-1",
    "iqn.2025-01.com.example:k8s-worker-2",
    "iqn.2025-01.com.example:k8s-worker-3",
  ]
}

# 3. Parent dataset for iSCSI volumes
resource "truenas_dataset" "k8s_iscsi" {
  pool        = "tank"
  name        = "k8s-iscsi"
  compression = "LZ4"
  comments    = "Parent dataset for Kubernetes iSCSI PVCs"
}
```

### Per-Application iSCSI LUN

For applications that need dedicated block devices (e.g., PostgreSQL):

```terraform
# Create a zvol for the database
resource "truenas_dataset" "postgres_zvol" {
  pool = "tank"
  name = "postgres-data"
  type = "VOLUME"
}

# Create the extent (LUN) backed by the zvol
resource "truenas_iscsi_extent" "postgres" {
  name    = "postgres-data"
  type    = "DISK"
  disk    = "zvol/${truenas_dataset.postgres_zvol.id}"
  comment = "PostgreSQL data volume"

  blocksize = 4096
  rpm       = "SSD"
}

# Create the target
resource "truenas_iscsi_target" "postgres" {
  name  = "iqn.2025-01.com.example:postgres-data"
  alias = "postgres-data"

  groups {
    portal    = truenas_iscsi_portal.k8s.tag
    initiator = tonumber(truenas_iscsi_initiator.k8s_nodes.id)
  }
}
```

## Snapshot Strategy for Kubernetes Workloads

```terraform
# Hourly snapshots of all k8s datasets, kept for 24 hours
resource "truenas_snapshot_task" "k8s_hourly" {
  dataset = "tank/k8s-nfs"

  recursive      = true
  lifetime_value = 24
  lifetime_unit  = "HOUR"

  schedule_minute = "0"
  schedule_hour   = "*"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"

  naming_schema = "hourly-%Y-%m-%d_%H-%M"
}

# Daily snapshots kept for 30 days
resource "truenas_snapshot_task" "k8s_daily" {
  dataset = "tank/k8s-nfs"

  recursive      = true
  lifetime_value = 30
  lifetime_unit  = "DAY"

  schedule_minute = "0"
  schedule_hour   = "2"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"

  naming_schema = "daily-%Y-%m-%d"
}
```

## truenas-csi Configuration

After provisioning the parent datasets with Terraform, point truenas-csi at the same pool and dataset paths. The driver reads its connection details from a ConfigMap and the API key from a Secret, then you define one StorageClass per protocol on the `csi.truenas.io` provisioner.

Connection:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: truenas-csi-config
  namespace: truenas-csi
data:
  # WebSocket JSON-RPC endpoint. The legacy REST API (/api/v2.0) is removed in TrueNAS 26.
  truenasURL: "wss://truenas.example.com/api/current"
  truenasInsecure: "false"
  defaultPool: "tank"
  nfsServer: "10.0.0.10"
  iscsiPortal: "10.0.0.10:3260"
  # Must match the TrueNAS global iSCSI base name (Shares > iSCSI > Global).
  iscsiIQNBase: "iqn.2005-10.org.example:k8s"
---
apiVersion: v1
kind: Secret
metadata:
  name: truenas-api-credentials
  namespace: truenas-csi
type: Opaque
stringData:
  api-key: "${TRUENAS_API_KEY}"
```

StorageClasses, one per protocol. `datasetPath` must match the parent dataset Terraform created above; truenas-csi creates a child dataset (plus the share or zvol) under it for each PVC:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: truenas-nfs
provisioner: csi.truenas.io
reclaimPolicy: Retain
allowVolumeExpansion: true
parameters:
  protocol: nfs
  datasetPath: tank/k8s-nfs
  compression: ZSTD
  sync: STANDARD
  nfs.networks: "10.0.0.0/24"
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: truenas-iscsi
provisioner: csi.truenas.io
reclaimPolicy: Retain
allowVolumeExpansion: true
parameters:
  protocol: iscsi
  datasetPath: tank/k8s-iscsi
  compression: LZ4
  sync: STANDARD
  volblocksize: "16K"
  csi.storage.k8s.io/fstype: ext4
```

## Validation

After applying your Terraform configuration, verify:

```bash
# Check the dataset was created
terraform output

# Check NFS share is active from a Kubernetes node
showmount -e 10.0.0.10

# Check iSCSI portal is reachable
iscsiadm -m discovery -t st -p 10.0.0.10:3260
```
