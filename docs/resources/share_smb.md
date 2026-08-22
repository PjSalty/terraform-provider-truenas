---
page_title: "truenas_share_smb Resource - terraform-provider-truenas"
subcategory: "Sharing"
description: |-
  Manages an SMB share on TrueNAS SCALE.
---

# truenas_share_smb (Resource)

Manages an SMB share on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
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
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path to share (e.g., /mnt/tank/data).
* `name` - (Required) The share name visible to SMB clients.
* `comment` - (Optional) A comment describing the share.
* `browsable` - (Optional) Whether the share is browsable in network discovery. Default: `true`.
* `readonly` - (Optional) Whether the share is read-only. Default: `false`.
* `abe` - (Optional) Whether Access Based Share Enumeration is enabled. Default: `false`.
* `enabled` - (Optional) Whether the share is enabled. Default: `true`.
* `purpose` - (Optional) The share purpose preset. Valid values: `DEFAULT_SHARE`, `LEGACY_SHARE`, `TIMEMACHINE_SHARE`, `MULTIPROTOCOL_SHARE`, `TIME_LOCKED_SHARE`, `PRIVATE_DATASETS_SHARE`, `EXTERNAL_SHARE`, `VEEAM_REPOSITORY_SHARE`, `FCP_SHARE`. Default: `DEFAULT_SHARE`.

  The pre-25.10 vocabulary (`ENHANCED_TIMEMACHINE`, `LEGACY_SMB_WHITELIST`, `MULTI_PROTOCOL_NFS`, `MULTI_PROTOCOL_AFP`, `PRIVATE_DATASETS`, `NO_PRESET`, `TIMEMACHINE`) was retired in the SMB preset overhaul and is no longer accepted. `FCP_SHARE` arrived in 25.10.1, so a 25.10.0 server rejects it.
* `options` - (Optional) Purpose-specific settings. See [below](#options).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Options

TrueNAS models `options` as a union keyed on `purpose`. An attribute belonging to
a different purpose is **rejected, not ignored**, so the provider checks the pairing
at plan time rather than letting the API return `Extra inputs are not permitted`.

`options` is a nested attribute, so it takes assignment syntax: `options = { ... }`.

| Attribute | Type | Purposes |
|---|---|---|
| `remote_path` | list(string) | `EXTERNAL_SHARE` (required) |
| `hostsallow` | list(string) | all except `EXTERNAL_SHARE` |
| `hostsdeny` | list(string) | all except `EXTERNAL_SHARE` |
| `aapl_name_mangling` | bool | `DEFAULT_SHARE`, `LEGACY_SHARE`, `MULTIPROTOCOL_SHARE`, `TIME_LOCKED_SHARE`, `PRIVATE_DATASETS_SHARE`, `FCP_SHARE` |
| `timemachine_quota` | number | `TIMEMACHINE_SHARE`, `LEGACY_SHARE` |
| `auto_snapshot` | bool | `TIMEMACHINE_SHARE` |
| `auto_dataset_creation` | bool | `TIMEMACHINE_SHARE` |
| `dataset_naming_schema` | string | `TIMEMACHINE_SHARE`, `PRIVATE_DATASETS_SHARE` |
| `auto_quota` | number | `PRIVATE_DATASETS_SHARE` |
| `grace_period` | number | `TIME_LOCKED_SHARE` |
| `vuid` | string | `TIMEMACHINE_SHARE`, `LEGACY_SHARE` |

Notes:

* `remote_path` - DFS proxy targets, each written `SERVER\SHARE`, where SERVER is a
  full domain name or an IP. HCL escapes backslashes, so write `"SERVER\\SHARE"` in
  the configuration. TrueNAS does not check that they are reachable.
  `EXTERNAL_SHARE` also requires `path = "EXTERNAL"`; it redirects clients rather
  than serving local storage.
* `aapl_name_mangling` - stores the illegal-NTFS characters macOS clients use with
  their native values. Do not change it once data is written; non-macOS clients may
  not see affected files. `FCP_SHARE` forces it on.
* `timemachine_quota` - bytes reported to the client for one sparsebundle, `0` for no
  quota. Modern macOS sets this client-side, which behaves more predictably.
* `dataset_naming_schema` - for example `%D/%U`. Unset means `%U`, or `%D/%U` when the
  server is joined to Active Directory. ZFS naming rules are stricter than path rules.
* `auto_quota` - gibibytes applied to each new dataset, `0` for none.
* `grace_period` - seconds a new file or directory stays writable, between `60` and
  `15552000` (180 days). Default `900`.
* `vuid` - volume UUID advertised to clients.

Every `options` attribute is also Computed: leave one out and the server's default for
that purpose is read back into state.

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_share_smb` resource.

## Import

The `truenas_share_smb` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_share_smb.example 1
```
