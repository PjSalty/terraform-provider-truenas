---
page_title: "truenas_init_script Resource - terraform-provider-truenas"
subcategory: "Scheduling"
description: |-
  Manages an init/startup script on TrueNAS SCALE.
---

# truenas_init_script (Resource)

Manages an init/startup script on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_init_script" "example" {
  type    = "COMMAND"
  when    = "POSTINIT"
  command = "/usr/bin/logger truenas init script executed"
  enabled = true
  comment = "Example init script"
  timeout = 10
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) Script type: COMMAND or SCRIPT. Valid values: `COMMAND`, `SCRIPT`.
* `when` - (Required) When to run: PREINIT, POSTINIT, or SHUTDOWN. Valid values: `PREINIT`, `POSTINIT`, `SHUTDOWN`.
* `command` - (Optional) The command to execute (when type is COMMAND).
* `script` - (Optional) The script path (when type is SCRIPT).
* `enabled` - (Optional) Whether the script is enabled. Default: `true`.
* `timeout` - (Optional) Timeout in seconds for the script. Default: `10`.
* `comment` - (Optional) A comment for the script.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_init_script` resource.

## Import

The `truenas_init_script` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_init_script.example 1
```
