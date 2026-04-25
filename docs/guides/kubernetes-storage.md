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
| NFS | ReadWriteMany (RWX) volumes, shared storage | `nfs.csi.k8s.io` or `democratic-csi` |
| iSCSI | ReadWriteOnce (RWO) volumes, databases | `org.democratic-csi.iscsi` |

## NFS Storage for Kubernetes (democratic-csi)

The recommended approach is to use [democratic-csi](https://github.com/democratic-csi/democratic-csi) with TrueNAS as the backend. Terraform provisions the datasets and NFS shares; democratic-csi dynamically creates sub-datasets for each PVC.

### Provision the Parent Dataset

```terraform
resource "truenas_dataset" "k8s_nfs" {
  pool        = "tank"
  name        = "k8s-nfs"
  compression = "LZ4"
  atime       = "OFF"
  share_type  = "GENERIC"
  comments    = "Parent dataset for Kubernetes NFS PVCs via democratic-csi"
}

resource "truenas_share_nfs" "k8s_nfs" {
  path     = truenas_dataset.k8s_nfs.mount_point
  comment  = "Kubernetes NFS PVC storage (democratic-csi)"
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

## Democratic-CSI Configuration

After provisioning TrueNAS resources with Terraform, configure democratic-csi with the matching dataset paths. An example `values.yaml` for the Helm chart:

```yaml
driver:
  config:
    driver: freenas-api-nfs
    httpConnection:
      protocol: https
      host: truenas.example.com
      apiKey: "${TRUENAS_API_KEY}"
    zfs:
      datasetParentName: tank/k8s-nfs
      detachedSnapshotsDatasetParentName: tank/k8s-nfs-snapshots
      datasetEnableQuotas: true
      datasetEnableReservation: false
      datasetPermissionsMode: "0777"
      datasetPermissionsUser: 0
      datasetPermissionsGroup: 0
    nfs:
      shareHost: 10.0.0.10
      shareAlldirs: false
      shareAllowedHosts: []
      shareAllowedNetworks: ["10.0.0.0/24"]
      shareMaprootUser: root
      shareMaprootGroup: wheel
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
