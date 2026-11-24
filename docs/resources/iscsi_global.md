---
page_title: "truenas_iscsi_global Resource - terraform-provider-truenas"
subcategory: "iSCSI"
description: |-
  Manages the iSCSI global configuration on TrueNAS SCALE. Singleton resource.
---

# truenas_iscsi_global (Resource)

Manages the iSCSI global configuration: the prefix of every target IQN, the iSNS
servers the targets register with, the port the iSCSI service listens on, and
the pool available space threshold.

This resource is a **singleton**, TrueNAS has exactly one. In the TrueNAS UI it
is the **Global Configuration** of **Shares > Block (iSCSI) Shares Targets**.

## Example Usage

```hcl
resource "truenas_iscsi_global" "this" {
  # Every target IQN is "<basename>:<target name>".
  basename = "iqn.2005-10.org.freenas.ctl"

  # IP addresses with an optional port (3205 by default), not hostnames.
  # TrueNAS must be able to connect to each one when this is applied.
  isns_servers = ["10.0.0.5"]

  listen_port = 3260

  # Pool available space threshold in percent, 1 to 99. 0 turns it off.
  pool_avail_threshold = 80
}
```

## Argument Reference

Every argument is optional. Anything the configuration does not set is left
untouched on the server, matching the update semantics of the underlying API,
and is read back into state.

* `basename` - (Optional) Prefix of every target IQN. TrueNAS defaults to
  `iqn.2005-10.org.freenas.ctl`. Must not be empty.
* `isns_servers` - (Optional) Set of iSNS servers, each an IP address with an
  optional `:port` (`3205` when none is given). Hostnames are not accepted. An
  empty set removes every server. See [iSNS servers](#isns-servers).
* `listen_port` - (Optional) TCP port the iSCSI service listens on, `1025` to
  `65535`. TrueNAS defaults to `3260`.
* `pool_avail_threshold` - (Optional) Pool available space threshold as a
  percentage, `1` to `99`. `0` turns it off. See
  [The threshold and 0](#the-threshold-and-0).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

* `id` - Fixed singleton identifier. Always `"iscsi_global"`.
* `alua` - Whether ALUA is enabled. Read-only, see [ALUA and iSER](#alua-and-iser).
* `iser` - Whether iSER is enabled. Read-only, see [ALUA and iSER](#alua-and-iser).

## Import

```shell
terraform import truenas_iscsi_global.this iscsi_global
```

The ID is the literal string `iscsi_global`; any other value is rejected.

## Behavior notes

### What an update does to the service

Every update reloads the iSCSI service. TrueNAS starts the reload and does not
wait for it, so an apply can finish before the reload has. Changing `iser`
(outside this resource) restarts the service if it is running.

Changing `basename` renames every target IQN, and changing `listen_port` moves
the port the service listens on, so either affects connected initiators. Plan
these changes for a maintenance window.

### iSNS servers

`isns_servers` is a set, and the provider sends it sorted. TrueNAS keeps the
servers in the order they were sent and the order has no meaning, so a set
never shows a difference that only reorders them.

On every update TrueNAS opens a TCP connection to each server in the request
and refuses the update if one does not answer. The provider sends the
servers on every update, including one that changes only `basename`, because
`iscsi.global.update` replaces the stored servers with the ones in the
request and a request without them would empty the list. An iSNS server that
is down therefore blocks every change to this resource, not only changes to
the servers.

### The threshold and 0

TrueNAS stores a switched-off `pool_avail_threshold` as null. A Terraform
number cannot be both set and null while keeping a plan that settles, so `0`
stands for null: it is sent as null and a null is read back as `0`. The same
convention is used for other nullable numbers in this provider. To remove a
threshold, set it to `0`; leaving the argument out keeps whatever is stored.

### ALUA and iSER

`alua` and `iser` are attributes you can read, not arguments. TrueNAS only
honors ALUA on Enterprise-licensed high availability systems, and on any
other system it stores the flag and ignores it. iSER needs an Enterprise
license and an RDMA-capable network card, and TrueNAS rejects enabling it
anywhere else. Exposing them as arguments would let a configuration set a
value that does nothing, or break the apply, on a system without the
hardware. If another tool enables them, the new value shows up in state with
no plan difference.

### Delete is a no-op

Removing this resource from configuration changes nothing on TrueNAS. The
configuration is a singleton that always exists, so there is nothing to
delete; Terraform simply stops managing it. The defaults are not put back,
because that would rename every target IQN and could move the port under
connected initiators.
