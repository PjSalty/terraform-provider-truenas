---
page_title: "truenas_ssh_config Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages the SSH service configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.
---

# truenas_ssh_config (Resource)

Manages the SSH service configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_ssh_config" "this" {
  tcpport           = 22
  rootlogin         = false
  passwordauth      = false
  kerberosauth      = false
  tcpfwd            = false
  compression       = false
  sftp_log_level    = "ERROR"
  sftp_log_facility = "AUTH"
}
```

## Argument Reference

The following arguments are supported:

* `tcpport` - (Optional) TCP port for the SSH service. Default: `22`.
* `passwordauth` - (Optional) Allow password authentication. Default: `true`.
* `kerberosauth` - (Optional) Allow Kerberos authentication. Default: `false`.
* `tcpfwd` - (Optional) Allow TCP port forwarding. Default: `false`.
* `compression` - (Optional) Enable compression. Default: `false`.
* `sftp_log_level` - (Optional) SFTP log level. Default: ``.
* `sftp_log_facility` - (Optional) SFTP log facility. Default: ``.
* `weak_ciphers` - (Optional) List of weak ciphers to allow (e.g., AES128-CBC, NONE).
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_ssh_config` resource.

## Import

The `truenas_ssh_config` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_ssh_config.this singleton
```
