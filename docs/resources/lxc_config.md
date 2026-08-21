---
page_title: "truenas_lxc_config Resource - terraform-provider-truenas"
subcategory: "Virtualization"
description: |-
  Manages the system-wide LXC container configuration on TrueNAS SCALE. Singleton resource; requires TrueNAS 26.0 or newer.
---

# truenas_lxc_config (Resource)

Manages the system-wide LXC container configuration: the default pool for
container and image datasets, and the bridge and networks used for container
networking.

This resource is a **singleton**, TrueNAS has exactly one LXC configuration.

~> **Requires TrueNAS 26.0 or newer.** The `lxc` API namespace does not exist on
25.10 or earlier. Against an older server every operation fails with a
diagnostic naming the required version rather than an opaque method-not-found.

## Example Usage

```hcl
resource "truenas_lxc_config" "this" {
  preferred_pool = "tank"

  # Empty means TrueNAS manages and creates a bridge automatically.
  bridge = ""

  # These are the TrueNAS defaults, kept clear of the Docker address pools
  # (172.16.0.0/12, fdd0::/48) and the 10.x.x.0/24 ranges Incus generates.
  v4_network = "172.200.0.0/24"
  v6_network = "fd42:4c58:43ae::/64"
}
```

## Argument Reference

Every argument is optional. Anything the configuration does not set is left
untouched on the server, matching the update semantics of the underlying API,
rather than being cleared.

* `preferred_pool` - (Optional) Default pool used for container and image
  datasets. Empty is a real state: TrueNAS then has no default and container
  creation must name a pool.
* `bridge` - (Optional) Network bridge interface for container networking.
  Empty means TrueNAS manages and creates one automatically. A named interface
  is validated against `lxc.bridge_choices` at apply time, so a bridge that does
  not exist is rejected with the valid choices listed rather than failing
  server-side. `lxc.bridge_choices` also advertises the sentinel `"[AUTO]"`;
  writing that is rejected at plan time, see
  [The `[AUTO]` bridge sentinel](#the-auto-bridge-sentinel).
* `v4_network` - (Optional) IPv4 network CIDR for the container bridge network.
  TrueNAS defaults to `172.200.0.0/24`.
* `v6_network` - (Optional) IPv6 network CIDR for the container bridge network.
  TrueNAS defaults to `fd42:4c58:43ae::/64`.

Both networks are checked at plan time: they must parse as a CIDR of the right
address family and cover at least 4 addresses (a `/30` or larger for IPv4).
TrueNAS additionally rejects a network that overlaps the system's own static
IPs, which can only be checked server-side at apply time.

## Attribute Reference

* `id` - Fixed singleton identifier. Always `"lxc_config"`.

## Import

```shell
terraform import truenas_lxc_config.this lxc_config
```

The ID is the literal string `lxc_config`; any other value is rejected.

## Behavior notes

### The `[AUTO]` bridge sentinel

`lxc.bridge_choices` returns `"[AUTO]"` for "let TrueNAS create and manage the
bridge", so it is the natural thing to copy into a configuration. It does not
work, and the provider rejects it at plan time:

```
Error: Invalid Bridge

  with truenas_lxc_config.this,
  on main.tf line 3, in resource "truenas_lxc_config" "this":
   3:   bridge = "[AUTO]"

"[AUTO]" is how the API advertises the automatic bridge in lxc.bridge_choices,
but TrueNAS stores it as a null that reads back as an empty string, so a
configuration using it can never converge. Set bridge = "" (or omit it) to have
TrueNAS manage the bridge.
```

TrueNAS stores the sentinel as a null, which refreshes back as `""`, so the
configuration would never converge. The provider cannot quietly rewrite it
either: for an `Optional` + `Computed` attribute Terraform requires the planned
value to equal a non-null config value, and rewriting it is refused as
"provider produced invalid plan". Use `bridge = ""`, or omit the argument.

### Delete is a no-op

Removing this resource from configuration does not reset container networking.
The LXC configuration is a singleton that always exists, so there is nothing to
delete; Terraform simply stops managing it.
