---
page_title: "truenas_mail_config Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages the email/SMTP configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.
---

# truenas_mail_config (Resource)

Manages the email/SMTP configuration on TrueNAS SCALE. This is a singleton resource — only one instance can exist.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_mail_config" "this" {
  fromemail      = "truenas@example.com"
  fromname       = "TrueNAS"
  outgoingserver = "smtp.example.com"
  port           = 587
  security       = "TLS"
  smtp           = true
  user           = "truenas"
  pass           = "smtp-password"
}
```

## Argument Reference

The following arguments are supported:

* `fromemail` - (Optional) From email address.
* `fromname` - (Optional) From name. Default: ``.
* `outgoingserver` - (Optional) Outgoing SMTP server hostname or IP. Default: ``.
* `port` - (Optional) SMTP port. Default: `25`.
* `security` - (Optional) Email security setting (PLAIN, SSL, TLS). Valid values: `PLAIN`, `SSL`, `TLS`. Default: `PLAIN`.
* `smtp` - (Optional) Enable SMTP authentication. Default: `false`.
* `user` - (Optional) SMTP authentication username. Default: ``.
* `pass` - (Optional) SMTP authentication password. Default: ``. Marked sensitive.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_mail_config` resource.

## Import

The `truenas_mail_config` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_mail_config.this singleton
```
