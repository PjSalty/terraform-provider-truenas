---
page_title: "truenas_system_update Resource - terraform-provider-truenas"
subcategory: "System Configuration"
description: |-
  Manages the TrueNAS SCALE system update configuration: the nightly update check and the tracked update profile. Singleton resource; does not execute updates.
---

# truenas_system_update (Resource)

Manages the TrueNAS SCALE system update configuration: the nightly update
check and the update profile the system tracks. This resource is a
**singleton**, TrueNAS has exactly one update config per system.

It does **not** execute updates. Applying an update remains a separate manual
action outside Terraform's control (UI, API call, or an Ansible playbook). Use
this resource to pin a profile and disable the automatic check so that SCALE
updates never happen without a conscious action.

~> **Breaking change in provider v2.5.** This resource previously exposed
`auto_download` and `train`, backed by five `update.*` API methods that do not
exist in TrueNAS middleware and never did on any release this provider
supports. Any plan touching it failed. See
[issue #32](https://github.com/PjSalty/terraform-provider-truenas/issues/32).
It is now built on `update.config` / `update.update`:
`auto_download` became `autocheck`, and `train` became `profile` because
TrueNAS 26.0 replaced release trains with update profiles. Existing state is
migrated automatically by a schema upgrader; `autocheck` carries over, and
`profile` is left empty for the next refresh to fill because a stored train
name is not a valid profile.

## Example Usage

### Pin a profile and disable the automatic check (recommended for prod)

```hcl
resource "truenas_system_update" "this" {
  autocheck = false
  profile   = "MISSION_CRITICAL"
}
```

### Follow whatever profile the system already has

```hcl
resource "truenas_system_update" "this" {
  autocheck = false
}
```

Omitting `profile` preserves whatever the system has configured and reports it
as a computed attribute.

## Schema

### Optional

- `autocheck` (Boolean) Whether TrueNAS automatically checks for and downloads
  updates nightly. Defaults to `false`, the conservative value: with it
  disabled, updates never land on the system without an explicit operator
  action.
- `profile` (String) The update profile this system tracks, for example
  `GENERAL` or `MISSION_CRITICAL`. Validated against `update.profile_choices`
  at apply time, honoring the `available` flag, so an unselectable profile is
  rejected with the valid choices listed rather than failing server-side.
- `timeouts` (Block, Optional)

### Read-Only

- `id` (String) Fixed singleton identifier. Always `"system_update"`.
- `status` (String) Update subsystem status code reported by TrueNAS: `NORMAL`
  or `ERROR`.
- `current_version` (String) The version the system is currently running.
- `available_version` (String) The version available to update to, empty when
  none is offered.

## Import

```shell
terraform import truenas_system_update.this system_update
```

The ID is the literal string `system_update`; any other value is rejected.

## Behavior notes

### Delete is a no-op

Removing this resource from configuration does not reset the system's update
policy. The update config is a singleton that always exists, so there is
nothing to delete; Terraform simply stops managing it.

### Version compatibility

`update.config`, `update.update`, `update.profile_choices` and `update.status`
are present on TrueNAS 25.10, 26.0 and 27.0 alike, so one code path covers
every supported release.
