---
page_title: "truenas_container Resource - terraform-provider-truenas"
subcategory: "Virtualization"
description: |-
  Manages an LXC container on TrueNAS SCALE. Requires TrueNAS 26.0 or newer.
---

# truenas_container (Resource)

Manages an LXC container: the image it is built from, its init process, its
capability policy and its user-namespace ID mapping.

~> **Requires TrueNAS 26.0 or newer.** The `container` API namespace does not
exist on 25.10 or earlier. Against an older server every operation fails with a
diagnostic naming the required version rather than an opaque method-not-found.

Devices (NIC, USB, filesystem, GPU) are not configured here. They live in the
`container.device` namespace upstream and are excluded from the container create
model, so they are managed separately.

## Example Usage

```terraform
resource "truenas_container" "web" {
  name = "web"

  image = {
    name    = "alpine:3.21:amd64:default"
    version = "20260820_23:08"
  }

  description = "Front-end container"
  autostart   = true
  pool        = "tank"

  initenv = {
    TZ = "UTC"
  }

  idmap = {
    type = "ISOLATED"
  }
}
```

## Argument Reference

* `name` - (Required) Container name.
* `image` - (Required) Image to build the container from. Both fields come from
  `container.image.query_registry`. Changing this forces a new container.
  * `name` - (Required) Image name, for example `alpine:3.21:amd64:default`.
  * `version` - (Required) Image version, for example `20260820_23:08`.
* `pool` - (Optional) Pool hosting the container's root filesystem. Unset uses
  the `preferred_pool` from `truenas_lxc_config`. Validated against
  `container.pool_choices` before the image pull starts, so a bad pool fails
  immediately rather than minutes into the job. Changing this forces a new
  container.
* `uuid` - (Optional) Container UUID used by libvirt. Generated when unset.
  Changing this forces a new container.
* `description` - (Optional) Free-text description.
* `autostart` - (Optional) Start the container on boot. Defaults to `true`.
* `time` - (Optional) `LOCAL` or `UTC`. Defaults to `LOCAL`.
* `shutdown_timeout` - (Optional) Seconds to wait for a clean shutdown before
  killing the container. Between 5 and 300, defaults to `90`.
* `cpuset` - (Optional) Physical CPU numbers the container may be pinned to,
  for example `"0-3"`. Empty means no pinning.
* `init` - (Optional) Init process command line. Defaults to `/sbin/init`.
* `initdir` - (Optional) Init working directory. Empty uses the image default.
* `initenv` - (Optional) Map of environment variables for the init process.
* `inituser` - (Optional) Username the init process runs as.
* `initgroup` - (Optional) Group the init process runs as.
* `capabilities_policy` - (Optional) `DEFAULT`, `ALLOW` or `DENY`. See
  [Capabilities](#capabilities). Defaults to `DEFAULT`.
* `capabilities_state` - (Optional) Map of capability name to boolean,
  overriding `capabilities_policy` per capability.
* `idmap` - (Optional) User-namespace ID mapping. See
  [ID mapping](#id-mapping). Changing this forces a new container.
  * `type` - (Required) `DEFAULT` or `ISOLATED`.
  * `slice` - (Optional) `ISOLATED` only, between 1 and 999. Omit to have
    TrueNAS pick an unused one.

## Attribute Reference

* `id` - Numeric container ID.
* `dataset` - Dataset used as the container root filesystem.
* `default_network` - Bridge the container uses when no NIC device is attached.
  Empty once NIC devices exist, because the configuration is then on the
  devices.
* `status` - Runtime state.
  * `state` - `RUNNING`, `STOPPED` or `SUSPENDED`.
  * `pid` - Host PID of the init process while running, `0` otherwise.
    Informational only; do not use it to identify the init process.
  * `domain_state` - Domain state as reported by libvirt.

## Import

```shell
terraform import truenas_container.web 1
```

`image` and `pool` cannot be recovered by an import. Both exist only on the
API's create model, so no read returns them and an imported container has
neither in state. Both are also `RequiresReplace`, which would normally make the
first apply after an import destroy and recreate the container you just adopted.
The provider suppresses that one case: while state has no value to compare
against, the configured `image` and `pool` are accepted as-is. A later change to
either replaces normally.

Put the values the container was actually built from in your configuration.
Nothing validates them against reality, because nothing can.

## Behavior notes

### ID mapping

`idmap` decides how container UIDs map onto host UIDs, so it is the security
boundary of the container.

* `DEFAULT` applies the standard TrueNAS namespace: container UID 0 becomes host
  UID 2147000001, and every other UID is offset by the same amount.
* `ISOLATED` does the same but offsets by `2147000001 + 65536 * slice`, so no
  two containers share host UIDs. Give each container a distinct `slice`, or
  omit it and let TrueNAS choose.

The API accepts a third value, a null idmap, which applies no namespace at all:
container root **is** host root. This resource cannot express it. Omitting the
`idmap` block gives you the TrueNAS default mapping, not an unmapped container.

If a container is already unmapped upstream, `idmap` reads back as null rather
than being reported as `DEFAULT`, so it is visible in the plan instead of
looking safer than it is.

### Capabilities

`capabilities_policy` sets the baseline:

* `DEFAULT` drops `sys_module`, `sys_time`, `mknod`, `audit_control` and
  `mac_admin`.
* `ALLOW` keeps every capability. Combined with an unmapped idmap this is
  effectively an unconfined root process on the host.
* `DENY` drops all capabilities.

`capabilities_state` then flips individual capabilities on top of that baseline.

### This resource does not start or stop containers

`status` is read-only. Use `autostart` for boot behaviour; an ad-hoc start or
stop is an operational action, not desired state, so it is left to the TrueNAS
UI and API.

### Removing the resource removes the root filesystem

Destroying a container removes its root filesystem.

The provider never passes the API's `recursive` option, which would additionally
remove the container dataset's child datasets and snapshots, any clones of those
snapshots anywhere in the pool, and any holds on them. If you need that, remove
those objects deliberately rather than as a side effect of tearing down
Terraform-managed infrastructure.

`container.delete` changed shape during the 26.0 cycle: 26.0-BETA.1 takes the
container ID alone and returns directly, while later builds accept an options
argument (`force`, `recursive`) and run the removal as a job. The provider sends
the newer form and falls back to the older one when the server rejects the extra
argument, so both work. On a server that supports it, `force` is set, which stops
a running container so the removal can proceed; on 26.0-BETA.1 there is no such
option, and removing a running container fails with the server's own error
rather than being force-stopped.
