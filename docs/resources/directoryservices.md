---
page_title: "truenas_directoryservices Resource - terraform-provider-truenas"
subcategory: "Auth & Integration"
description: |-
  Manages the TrueNAS directory services singleton configuration (ACTIVEDIRECTORY, IPA, or LDAP). Only one directory service can be active at a time. Creating this resource configures the service; deleting it disables the service (it does not unjoin from AD — use the TrueNAS UI/CLI or `enable = false` first).
---

# truenas_directoryservices (Resource)

Manages the TrueNAS directory services singleton configuration (ACTIVEDIRECTORY, IPA, or LDAP). Only one directory service can be active at a time. Creating this resource configures the service; deleting it disables the service (it does not unjoin from AD — use the TrueNAS UI/CLI or `enable = false` first).

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
# Singleton — one directory services configuration per system.
resource "truenas_directoryservices" "this" {
  service_type = "ACTIVEDIRECTORY"
  enable       = true
  configuration_json = jsonencode({
    hostname           = "dc1.example.com"
    domainname         = "EXAMPLE.COM"
    bindname           = "administrator"
    bindpw             = "ChangeMe!2026"
    use_default_domain = false
  })
}
```

## Argument Reference

The following arguments are supported:

* `service_type` - (Optional) One of ACTIVEDIRECTORY, IPA, LDAP. Set to empty string to disable. Valid values: `, `, `, `, `, `.
* `enable` - (Optional) Whether the directory service is enabled. Default: `false`.
* `enable_account_cache` - (Optional) Enable backend caching for user and group lists. Default: `true`.
* `enable_dns_updates` - (Optional) Enable automatic DNS updates via nsupdate. Default: `true`.
* `timeout` - (Optional) Timeout (seconds) for DNS queries performed during domain join. Default: `10`.
* `kerberos_realm` - (Optional) Name of the Kerberos realm used for authentication. Required for ACTIVEDIRECTORY and IPA; optional for LDAP.
* `credential_json` - (Optional) JSON-encoded credential object. Shape depends on credential_type (KERBEROS_USER, KERBEROS_PRINCIPAL, LDAP_PLAIN, LDAP_MTLS, LDAP_ANONYMOUS). Example: jsonencode({ credential_type = "KERBEROS_USER", username = "admin", password = "..." }). Marked sensitive.
* `configuration_json` - (Optional) JSON-encoded service_type-specific configuration (domain, hostname, base_dn, server_urls, etc.). See the TrueNAS API docs for required fields.
* `force` - (Optional) Bypass validation that checks if a server with this hostname or NetBIOS name is already joined. Use with caution.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `10m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `10m`) Timeout for updating the resource.
* `delete` - (Default `10m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_directoryservices` resource.

## Import

The `truenas_directoryservices` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_directoryservices.this singleton
```
