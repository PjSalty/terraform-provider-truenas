---
page_title: "truenas_system_update Resource - terraform-provider-truenas"
subcategory: "System Configuration"
description: |-
  Manages the TrueNAS SCALE system update configuration: the auto-download toggle and the active release train. Singleton resource; does not execute updates.
---

# truenas_system_update (Resource)

Manages the TrueNAS SCALE system update configuration: the auto-download
toggle and the active release train. This resource is a **singleton** — TrueNAS
has exactly one update config per system.

It does **not** execute updates. Applying an update remains a separate manual
action outside Terraform's control (UI, API call, or an Ansible playbook).
Use this resource to pin a train and/or disable auto-download so that SCALE
updates never happen without a conscious action.

## Example Usage

### Pin the current train, disable auto-download (recommended for prod)

```terraform
resource "truenas_system_update" "prod" {
  auto_download = false
  train         = "TrueNAS-SCALE-Fangtooth"
}
```

### Let TrueNAS follow whatever train is currently selected

Omit `train` and the provider reads and preserves the system's existing
selection on every apply. `auto_download` still defaults to `false`.

```terraform
resource "truenas_system_update" "prod" {
  auto_download = false
}
```

## Schema

### Optional

- `auto_download` (Boolean) — Whether TrueNAS should automatically download
  available updates into the local update cache. Defaults to `false` — the
  conservative pinning value. With `auto_download` disabled, updates never
  land on the system without an explicit operator action.
- `train` (String) — The active release train (for example,
  `TrueNAS-SCALE-Fangtooth`). When set, Terraform reconciles the selected
  train on every apply. When omitted, Terraform preserves whatever the
  system has configured and reports it as a computed attribute. Validated
  against the list returned by the TrueNAS API at apply time.

### Read-Only

- `id` (String) — Fixed singleton identifier. Always `"system_update"`.
- `current_version` (String) — The version of TrueNAS SCALE currently
  running on the system. Refreshed from `/system/info` on every Read.
- `available_status` (String) — The pending-update status reported by the
  TrueNAS update server. One of `AVAILABLE`, `UNAVAILABLE`, `REBOOT_REQUIRED`,
  `HA_UNAVAILABLE`. `UNAVAILABLE` is the normal steady-state value.
- `available_version` (String) — When `available_status` is `AVAILABLE`,
  this is the version string of the pending update. Empty in all other
  states.

## Import

The resource is a singleton; the only valid import ID is the literal string
`system_update`.

```shell
terraform import truenas_system_update.prod system_update
```

## Behaviour notes

### Delete is a no-op

`terraform destroy` removes the resource from Terraform state but does **not**
reset the TrueNAS update config. The last-applied `auto_download` and `train`
values remain in effect on the system. This prevents a surprising reboot-risk
vector where an accidental `destroy` could re-enable auto-download and stage
an upgrade.

### Update execution is out of scope

This resource cannot start, cancel, or apply a pending update. Those actions
are intentionally kept outside Terraform so a routine `terraform plan` diff
can never end up rebooting production. To apply an update, use the TrueNAS
UI, an API call, or a dedicated Ansible playbook gated on backups and a
maintenance window.
