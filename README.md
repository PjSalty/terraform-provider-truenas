# Terraform Provider for TrueNAS SCALE

[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-brightgreen.svg)](LICENSE)
[![Terraform](https://img.shields.io/badge/terraform-%3E%3D1.0-623CE4)](https://www.terraform.io/)
[![TrueNAS SCALE](https://img.shields.io/badge/TrueNAS%20SCALE-24.04%2B-0095D5)](https://www.truenas.com/truenas-scale/)

Terraform provider for managing
[TrueNAS SCALE](https://www.truenas.com/truenas-scale/) storage, network,
and virtualization resources through the REST API v2.0. Built on
`terraform-plugin-framework`.

---

## Table of Contents

- [Installation](#installation)
- [Quickstart](#quickstart)
- [Authentication](#authentication)
- [Resources](#resources)
- [Data Sources](#data-sources)
- [Version Compatibility](#version-compatibility)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Installation

```hcl
terraform {
  required_providers {
    truenas = {
      source  = "PjSalty/truenas"
      version = "~> 0.4"
    }
  }
}

provider "truenas" {
  url     = "https://truenas.example.com"
  api_key = var.truenas_api_key
}
```

## Quickstart

```hcl
# Create a ZFS dataset.
resource "truenas_dataset" "media" {
  name        = "media"
  pool        = "tank"
  compression = "LZ4"
  quota       = 1099511627776 # 1 TiB
  comments    = "Media library"
}

# Share it over NFS.
resource "truenas_share_nfs" "media" {
  path     = "/mnt/tank/media"
  comment  = "Media NFS share"
  networks = ["10.0.0.0/24"]

  maproot_user  = "root"
  maproot_group = "wheel"
}

# Take hourly snapshots with 7-day retention.
resource "truenas_snapshot_task" "media_hourly" {
  dataset        = truenas_dataset.media.id
  recursive      = true
  lifetime_value = 7
  lifetime_unit  = "DAY"
  naming_schema  = "auto-%Y-%m-%d_%H-%M"

  schedule {
    minute = "0"
    hour   = "*"
    dom    = "*"
    month  = "*"
    dow    = "*"
  }
}
```

## Authentication

The provider authenticates to the TrueNAS REST API with an API key.

| Argument               | Environment Variable          | Required | Description                                                                                                                                            |
| ---------------------- | ----------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `url`                  | `TRUENAS_URL`                 | yes      | Base URL of the TrueNAS instance (HTTPS).                                                                                                              |
| `api_key`              | `TRUENAS_API_KEY`             | yes      | API key created under Credentials → API Keys.                                                                                                          |
| `insecure_skip_verify` | `TRUENAS_INSECURE_SKIP_VERIFY` | no       | Skip TLS verification for self-signed test environments. Never enable this against production.                                                        |
| `read_only`            | `TRUENAS_READONLY`            | no       | When true, the provider refuses every mutating request (POST/PUT/DELETE) before it reaches the network. See [Read-only mode](#read-only-mode-safety-rail). |

The provider normalizes the base URL and appends `/api/v2.0` automatically.
API keys should be stored in a secret manager (e.g. SOPS, Vault,
Vaultwarden) and injected via environment variables in CI.

```bash
export TRUENAS_URL="https://truenas.example.com"
export TRUENAS_API_KEY="1-abc123..."
```

### Read-only mode (safety rail)

For phased production rollout, set `read_only = true` (HCL) or
`TRUENAS_READONLY=1` (env). When enabled, every mutating request
(POST/PUT/DELETE) fails with `ErrReadOnly` **before any network call
is made** — the target TrueNAS instance never even sees the attempt,
not even in its access log.

```hcl
provider "truenas" {
  url       = "https://prod.truenas.example.com"
  api_key   = var.prod_key
  read_only = true  # safe to run `terraform plan` against prod
}
```

```bash
TRUENAS_READONLY=1 terraform plan
```

Intended use: point the provider at production, run `terraform plan`,
and be **physically incapable** of mutating anything. A surprised or
buggy plan surfaces as a normal Terraform error instead of a partial
write. Flip the flag off only once the plan shows exactly what you
expect to happen.

HCL takes precedence over the env var when both are set, matching
the provider's usual precedence rules.

### Destroy-protection mode (apply-safe rail)

For the first production apply, layer `destroy_protection = true`
(HCL) or `TRUENAS_DESTROY_PROTECTION=1` (env) on top. This is a
second safety rail that blocks ONLY `DELETE` requests at the client
layer — `GET`, `POST`, and `PUT` flow through normally. Create and
update work; destroy is physically refused:

```hcl
provider "truenas" {
  url                = "https://prod.truenas.example.com"
  api_key            = var.prod_key
  read_only          = false  # allow create/update
  destroy_protection = true   # but NO resource can be destroyed
}
```

This matches the AWS provider's per-resource `deletion_protection`
pattern (on `aws_db_instance`, `aws_eks_cluster`, `aws_lb`, etc.) —
except it's enforced at the wire for every resource in the provider
at once, so there is no per-resource coverage gap to audit.

Intended use: the first `terraform apply` against a real production
TrueNAS. Creates and updates work; a mis-typed removal in HCL cannot
destroy anything until the operator explicitly clears the flag. See
`docs/guides/phased-rollout.md` Phase 3 for the full drill.

If you need to run an intentional destroy:

```sh
unset TRUENAS_DESTROY_PROTECTION
terraform apply
export TRUENAS_DESTROY_PROTECTION=1  # re-arm immediately
```

### First prod rollout: the smoke workspace

`examples/prod-smoke/` is a committed, ready-to-run Terraform workspace
that imports one existing dataset from your production TrueNAS and
verifies the provider can refresh state without any ability to mutate
anything. Both safety rails are armed by default (`read_only = true` +
`destroy_protection = true`).

Copy it out of the repo, fill in two variables, and run:

```sh
cp -r examples/prod-smoke ~/tf-truenas-prod-smoke
cd ~/tf-truenas-prod-smoke
export TF_VAR_truenas_api_key="$(sops -d path/to/secrets.sops.yaml | yq -r '.truenas.api_key')"
export TF_VAR_smoke_dataset_pool="tank"
export TF_VAR_smoke_dataset_name="path/to/your/existing/dataset"
terraform plan
# Expected: Plan: 0 to add, 0 to change, 0 to destroy.
```

See `examples/prod-smoke/RUN.md` for the full runbook including the
Phase 2 (apply-safe) and Phase 3 (brief destroy window) transitions.

## Resources

Every resource ships with validators on each argument, round-trip tests,
generated docs under `docs/resources/`, and a runnable example under
`examples/resources/<name>/resource.tf`.

### Storage

| Resource                          | Purpose                                    |
| --------------------------------- | ------------------------------------------ |
| `truenas_pool`                    | ZFS pool lifecycle (create/export/import). |
| `truenas_dataset`                 | ZFS filesystem dataset.                    |
| `truenas_zvol`                    | ZFS volume (block device).                 |
| `truenas_snapshot_task`           | Periodic ZFS snapshot schedule.            |
| `truenas_scrub_task`              | Scheduled pool scrub.                      |
| `truenas_replication`             | ZFS send/recv replication task.            |
| `truenas_systemdataset`           | System dataset pool assignment.            |
| `truenas_filesystem_acl`          | POSIX/NFSv4 ACL on a filesystem path.      |
| `truenas_filesystem_acl_template` | Reusable ACL template.                     |

### Sharing

| Resource                    | Purpose                                  |
| --------------------------- | ---------------------------------------- |
| `truenas_share_nfs`         | NFSv3/v4 share.                          |
| `truenas_share_smb`         | SMB/CIFS share.                          |
| `truenas_iscsi_portal`      | iSCSI portal (listen address/port).      |
| `truenas_iscsi_initiator`   | iSCSI authorized initiator group.        |
| `truenas_iscsi_extent`      | iSCSI extent (LUN backing).              |
| `truenas_iscsi_target`      | iSCSI target.                            |
| `truenas_iscsi_targetextent`| Target→extent association (LUN binding).|
| `truenas_iscsi_auth`        | iSCSI CHAP auth network.                 |

### Networking

| Resource                     | Purpose                                     |
| ---------------------------- | ------------------------------------------- |
| `truenas_network_config`     | Global network configuration (DNS, proxy).  |
| `truenas_network_interface`  | Physical/bridge/VLAN/LAGG interface.        |
| `truenas_static_route`       | Static IP route.                            |
| `truenas_dns_nameserver`     | DNS nameserver entry (single-value resource).|

### Virtualization

| Resource             | Purpose                                           |
| -------------------- | ------------------------------------------------- |
| `truenas_vm`         | KVM virtual machine.                              |
| `truenas_vm_device`  | VM device (DISK, NIC, DISPLAY, USB, RAW, CDROM).  |
| `truenas_vmware`     | VMware snapshot integration credentials.          |
| `truenas_app`        | TrueCharts/SCALE Apps Helm release.               |
| `truenas_catalog`    | SCALE Apps catalog.                               |

### Identity & Access

| Resource                 | Purpose                              |
| ------------------------ | ------------------------------------ |
| `truenas_user`           | Local user.                          |
| `truenas_group`          | Local group.                         |
| `truenas_privilege`      | Role-based privilege assignment.     |
| `truenas_directoryservices` | LDAP/AD/IPA/Kerberos integration. |
| `truenas_kerberos_realm` | Kerberos realm definition.           |
| `truenas_kerberos_keytab`| Kerberos keytab upload.              |
| `truenas_api_key`        | API key provisioning.                |

### Data Protection

| Resource                    | Purpose                                |
| --------------------------- | -------------------------------------- |
| `truenas_cloud_sync`           | Cloud sync (S3/GCS/Azure/B2/...) task.                                 |
| `truenas_cloud_backup`         | Restic/rclone backup task.                                             |
| `truenas_cloudsync_credential` | Cloud-storage credential (S3, B2, Azure Blob, GCS, Dropbox, FTP, ...). |
| `truenas_rsync_task`           | Rsync push/pull task.                                                  |
| `truenas_kmip_config`          | KMIP key management integration.                                       |

### System

| Resource                   | Purpose                                       |
| -------------------------- | --------------------------------------------- |
| `truenas_cronjob`          | System cron job.                              |
| `truenas_init_script`      | Init/shutdown script hook.                    |
| `truenas_tunable`          | Kernel sysctl tunable.                        |
| `truenas_service`          | Service enable/start/stop.                    |
| `truenas_alert_service`    | Alert notification channel.                   |
| `truenas_alertclasses`     | Alert class severity/policy mapping.          |
| `truenas_reporting_exporter`| Prometheus/Graphite reporting exporter.      |

### Storage Protocols (config singletons)

| Resource             | Purpose                             |
| -------------------- | ----------------------------------- |
| `truenas_nfs_config` | Global NFS server configuration.    |
| `truenas_smb_config` | Global SMB server configuration.    |
| `truenas_ftp_config` | Global FTP server configuration.    |
| `truenas_ssh_config` | Global SSH server configuration.    |
| `truenas_snmp_config`| Global SNMP agent configuration.    |
| `truenas_ups_config` | UPS (NUT) monitoring configuration. |
| `truenas_mail_config`| Outgoing mail (SMTP) configuration. |

### NVMe-oF (Subsystem)

| Resource                    | Purpose                              |
| --------------------------- | ------------------------------------ |
| `truenas_nvmet_global`      | NVMe-oF global settings.             |
| `truenas_nvmet_host`        | Allowed host NQN.                    |
| `truenas_nvmet_subsys`      | Subsystem definition.                |
| `truenas_nvmet_port`        | Transport port (TCP/RDMA).           |
| `truenas_nvmet_namespace`   | Namespace backing (zvol/file).       |
| `truenas_nvmet_host_subsys` | Host→subsys binding.                 |
| `truenas_nvmet_port_subsys` | Port→subsys binding.                 |

### Certificates

| Resource                        | Purpose                                   |
| ------------------------------- | ----------------------------------------- |
| `truenas_certificate`           | TLS certificate (import/CSR/CA-signed).   |
| `truenas_acme_dns_authenticator`| ACME DNS-01 authenticator.                |
| `truenas_keychain_credential`   | SSH keypair / SSH connection credential.  |

## Data Sources

| Data Source                  | Description                                       |
| ---------------------------- | ------------------------------------------------- |
| `truenas_dataset`            | Read a ZFS dataset by path.                       |
| `truenas_datasets`           | List datasets (optionally filtered by pool).      |
| `truenas_pool`               | Read a pool by name/ID.                           |
| `truenas_pools`              | List all pools.                                   |
| `truenas_disk`               | Read a disk by identifier.                        |
| `truenas_system_info`        | Read live system info (version, CPU, mem).        |
| `truenas_network_config`     | Read global network config.                       |
| `truenas_network_interface`  | Read an interface by name.                        |
| `truenas_systemdataset`      | Read the system dataset pool.                     |
| `truenas_user`               | Read a user by username/ID.                       |
| `truenas_group`              | Read a group by name/ID.                          |
| `truenas_privilege`          | Read a privilege by name/ID.                      |
| `truenas_kerberos_realm`     | Read a Kerberos realm.                            |
| `truenas_directoryservices`  | Read directory services config.                   |
| `truenas_app`                | Read a SCALE App by name.                         |
| `truenas_apps`               | List deployed SCALE Apps.                         |
| `truenas_catalog`            | Read a SCALE Apps catalog.                        |
| `truenas_vm`                 | Read a VM by name/ID.                             |
| `truenas_vms`                | List all VMs.                                     |
| `truenas_share_nfs`          | Read an NFS share by path/ID.                     |
| `truenas_share_smb`          | Read an SMB share by name/ID.                     |
| `truenas_cronjob`            | Read a cron job by ID.                            |
| `truenas_service`            | Read a service by name.                           |

Every data source has generated docs under `docs/data-sources/` and an
example under `examples/data-sources/<name>/data-source.tf`.

## Version Compatibility

| Provider Version | TrueNAS SCALE | Terraform | Go (build) |
| ---------------- | ------------- | --------- | ---------- |
| `0.4.x`          | 24.04 – 25.10 | `>= 1.0`  | `>= 1.23`  |
| `0.3.x`          | 24.04 – 25.04 | `>= 1.0`  | `>= 1.23`  |
| `0.1.x`          | 24.04         | `>= 1.0`  | `>= 1.22`  |

SCALE 25.10 introduced several schema-breaking changes (alert services,
dataset comments) that the 0.4.x provider handles transparently; older
provider versions will surface spurious drift when pointed at a 25.10
instance.

## Development

### Prerequisites

- Go `>= 1.23`
- Terraform `>= 1.0`
- A TrueNAS SCALE test instance reachable from your machine

### Build

```bash
go build ./...
```

### Test

```bash
# Unit tests (no TrueNAS instance required).
go test ./...

# Acceptance tests (requires TRUENAS_URL and TRUENAS_API_KEY).
TF_ACC=1 go test ./internal/resources/ -v -timeout 30m
```

### Generate docs

```bash
go generate ./...
```

Docs are generated into `docs/` from provider schemas and example files
via [`terraform-plugin-docs`](https://github.com/hashicorp/terraform-plugin-docs).

### Local install (development overrides)

Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "PjSalty/truenas" = "/path/to/terraform-provider-truenas"
  }
  direct {}
}
```

Then run `go build -o terraform-provider-truenas` in this repo and any
`terraform plan` in a configuration that uses the provider will pick up
your local binary.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the contribution workflow,
coding standards, and testing expectations. In short:

1. Open an issue describing the change.
2. Create a feature branch.
3. Add validators, docs, examples, and round-trip tests for any new
   resource.
4. Run `go test ./...`, `go vet ./...`, and `gofmt -l .` — all must be
   clean.
5. Open a merge request.

## License

This provider is distributed under the terms of the Mozilla Public
License 2.0. See [`LICENSE`](LICENSE) for the full text.
