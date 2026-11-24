---
page_title: "truenas_replication_config Resource - terraform-provider-truenas"
subcategory: "Scheduling"
description: |-
  Manages the replication global configuration on TrueNAS SCALE. Singleton resource.
---

# truenas_replication_config (Resource)

Manages the replication global configuration: how many replication tasks may
run at the same time. Tasks beyond the limit wait until a running one finishes.

This resource is a **singleton**, TrueNAS has exactly one. In the TrueNAS UI it
is the **Settings** of **Data Protection > Replication Tasks**.

## Example Usage

```hcl
resource "truenas_replication_config" "this" {
  # At least 1. TrueNAS defaults to 5. 0 means no limit.
  max_parallel_replication_tasks = 3
}
```

## Argument Reference

Every argument is optional. Anything the configuration does not set is left
untouched on the server, matching the update semantics of the underlying API,
and is read back into state.

* `max_parallel_replication_tasks` - (Optional) Maximum number of replication
  tasks that run in parallel, at least `1`. `0` means no limit. TrueNAS
  defaults to `5`. See [No limit](#no-limit).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

* `id` - Fixed singleton identifier. Always `"replication_config"`.

## Import

```shell
terraform import truenas_replication_config.this replication_config
```

The ID is the literal string `replication_config`; any other value is rejected.

## Behavior notes

### No limit

TrueNAS stores "no limit" as null. A Terraform number cannot be both set and
null while keeping a plan that settles, so `0` stands for null: it is sent as
null and a null is read back as `0`. The same convention is used for other
nullable numbers in this provider. To remove a limit, set the argument to `0`;
leaving it out keeps whatever is stored, so a configuration that stops
mentioning the limit does not remove it.

### When a change takes effect

A change is passed to the running replication engine as soon as it is saved.
Nothing is restarted, and tasks that are already running are not stopped.

### Delete is a no-op

Removing this resource from configuration changes nothing on TrueNAS. The
configuration is a singleton that always exists, so there is nothing to
delete; Terraform simply stops managing it. The limit is left as it is,
because raising or removing it would start replication tasks that were being
held back.
