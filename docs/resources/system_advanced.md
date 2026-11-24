---
page_title: "truenas_system_advanced Resource - terraform-provider-truenas"
subcategory: "System"
description: |-
  Manages the System > Advanced settings on TrueNAS SCALE. Singleton resource.
---

# truenas_system_advanced (Resource)

Manages the **System > Advanced** settings: remote syslog, the message of the
day and login banner, the console, boot and crash settings. TrueNAS has exactly
one set of these, so this resource is a **singleton**.

Remote syslog is the main use. Several other settings act on the running
system when they are applied; the [Behavior notes](#behavior-notes) list what
each one does.

## Example Usage

```hcl
resource "truenas_system_advanced" "this" {
  syslogservers = [
    { host = "192.0.2.10" },                               # UDP, port 514
    { host = "logs.example.com:6514", transport = "TLS" }, # one-way TLS
  ]
  sysloglevel  = "F_INFO"
  syslog_audit = true

  motd         = "Managed by Terraform"
  login_banner = "Authorised use only."
  boot_scrub   = 7
}
```

## Argument Reference

Every argument is optional. Anything the configuration does not set is left
untouched on the server, matching the update semantics of the underlying API,
and is read back into state.

### Remote syslog

* `syslogservers` - (Optional) Ordered list of at most two remote syslog
  servers. Setting it replaces the whole list, and `[]` removes every server.
  See [Remote syslog](#remote-syslog-1).
  * `host` - (Required) DNS name or IP address, with an optional `:port`.
    Without a port TrueNAS uses 514 for UDP and TCP, and 6514 for TLS. An IPv6
    address must be in brackets: `[2001:db8::10]` or `[2001:db8::10]:5514`.
    Hosts must be unique within the list.
  * `transport` - (Optional) `UDP` (the default), `TCP` or `TLS`.
  * `tls_certificate` - (Optional) ID of a local certificate, sent for mutual
    TLS. Only valid with `transport = "TLS"`; leave it out for one-way TLS.
* `sysloglevel` - (Optional) Minimum severity: `F_EMERG`, `F_ALERT`,
  `F_CRIT`, `F_ERR`, `F_WARNING`, `F_NOTICE`, `F_INFO` or `F_DEBUG`. TrueNAS
  defaults to `F_INFO`.
* `fqdn_syslog` - (Optional) Write the full domain name into syslog messages.
  Defaults to `false`.
* `syslog_audit` - (Optional) Also send audit messages to the remote servers.
  Has no effect while `syslogservers` is empty. Defaults to `false`.

### Login and message text

* `motd` - (Optional) Message of the day, at most 1024 characters. TrueNAS
  defaults to `Welcome to TrueNAS`.
* `login_banner` - (Optional) Banner shown before the login prompt, at most
  4096 characters. An empty string means no banner.

### Console and boot

* `consolemenu` - (Optional) Show the console menu on the local console
  instead of a plain login prompt.
* `serialconsole` - (Optional) Enable a login on a serial port. Defaults to
  `false`.
* `serialport` - (Optional) Serial port device, such as `ttyS0`. Defaults to
  `ttyS0`.
* `serialspeed` - (Optional) Baud rate: `9600`, `19200`, `38400`, `57600` or
  `115200`. Defaults to `9600`.
* `kernel_extra_options` - (Optional) Extra kernel command line parameters,
  at most 1024 characters. Defaults to an empty string.
* `kdump_enabled` - (Optional) Enable kernel crash dumps. Defaults to `false`.
* `debugkernel` - (Optional) Boot the debug kernel. Defaults to `false`.
* `boot_scrub` - (Optional) Days between boot pool scrubs, at least 1.
  Defaults to `7`.
* `powerdaemon` - (Optional) Run the power management daemon. Defaults to
  `false`.
* `overprovision` - (Optional) Percentage of an SSD to leave unused when a
  pool with log devices is created. `0` disables it.

### Stored settings

* `traceback`, `uploadcrash`, `anonstats` - (Optional) Booleans TrueNAS stores
  for the web interface and for crash and usage reporting. Each defaults to
  `true`. The 25.10 middleware only stores them: applying one does nothing
  else.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

* `id` - Fixed singleton identifier. Always `"system_advanced"`.

## Import

```shell
terraform import truenas_system_advanced.this system_advanced
```

The ID is the literal string `system_advanced`; any other value is rejected.

## Behavior notes

### Remote syslog

* TrueNAS stores `syslogservers` as a list and reads it back in the order it
  was written, so the attribute is an ordered list and a reordering is a
  change. The order only decides which server comes first in the generated
  syslog-ng configuration.
* An empty list removes every server. Leaving the argument out keeps them.
* `UDP` and `TCP` send in clear text. `TLS` verifies the server against the
  system CA bundle (`/etc/ssl/certs/ca-certificates.crt`); setting
  `tls_certificate` adds a client certificate for mutual TLS. TrueNAS sets
  `tls_certificate` to null for UDP and TCP, so the provider rejects a
  certificate there instead of letting the plan diverge.
* The provider checks the host at plan time the way TrueNAS does (a name or
  address with an optional numeric `:port` from 0 to 65535, unique within the
  list). It also rejects an unbracketed IPv6 address, which TrueNAS would read
  as a host and a port, and a non-numeric port, which makes it fail with an
  internal error.
* `syslog_audit` only sends audit messages when at least one server is set.

### What applying each setting does

`system.advanced.update` is one call, and TrueNAS does the following before it
returns. Nothing happens for a setting whose value did not change.

| Setting | What applying a change does |
| --- | --- |
| `syslogservers`, `sysloglevel`, `fqdn_syslog`, `syslog_audit` | Restarts the syslog service. |
| `boot_scrub` | Restarts cron, which holds the boot pool scrub job. |
| `motd` | Regenerates `/etc/motd`. |
| `login_banner` | Reloads the SSH service. |
| `powerdaemon` | Restarts the power daemon. |
| `consolemenu` | Restarts `getty@tty1`, which ends the session open on the local console. |
| `serialconsole`, `serialport`, `serialspeed` | Enables, disables or restarts the `serial-getty` unit and regenerates the GRUB configuration. |
| `kernel_extra_options` | Regenerates the GRUB configuration. Takes effect at the next restart of the machine. A wrong value can stop the system booting. |
| `kdump_enabled` | Regenerates the kdump and GRUB configuration. Takes effect at the next restart of the machine. |
| `debugkernel` | Regenerates the GRUB configuration, and rebuilds the initramfs when switched on. Takes effect at the next restart of the machine. |
| `overprovision` | Nothing at once. Used when a pool with log devices is created. |
| `traceback`, `uploadcrash`, `anonstats` | Nothing. |

If one of those service restarts fails, TrueNAS has already saved the new
values and returns an error: Terraform reports the failure, and the next
refresh shows the stored values.

TrueNAS does not restart the machine itself. The settings marked as taking
effect at the next restart do nothing until then.

### Checks only TrueNAS can make

* `kernel_extra_options` is checked at plan time for a newline, a double quote
  and `systemd.unified_cgroup_hierarchy`. TrueNAS also runs `grub-script-check`
  on it, which fails at apply time for anything GRUB cannot parse.
* With `serialconsole = true`, `serialport` must be a port TrueNAS has found
  and must differ from the UPS service's port. Both are checked at apply time.
* Only the login banner is limited to 4096 characters. TrueNAS limits every
  other string to 1024.

### `overprovision` cannot return to null

A fresh TrueNAS has `overprovision` unset (null). The API takes a number or
null, but the provider cannot send null for an argument left out of the
configuration, so to clear it set `0`, which TrueNAS treats the same way, and
which then reads back as `0`.

### Delete is a no-op

Removing this resource does not put the defaults back. The configuration always
exists, so there is nothing to delete; Terraform simply stops managing it. A
destroy that dropped the remote syslog servers, or changed the kernel command
line or the serial console, would be a surprise, and the last two only show up
at the next restart. Set the values you want before removing the resource.

### Settings this resource does not manage

* `sed_user` and `sed_passwd`: the global password for self-encrypting drives
  is a write-only secret and belongs in a resource of its own.
* `anonstats_token`: read-only.
* `isolated_gpu_pci_ids`: written through a separate method,
  `system.advanced.update_gpu_pci_ids`.
* `consolemsg`: deprecated here, TrueNAS moved it to the general system
  settings (`ui_consolemsg`).
* `advancedmode`: a display preference of the web interface that the middleware
  never reads.
* `autotune`: nothing in the 25.10 middleware reads it, apart from the
  Enterprise first-boot code that sets it.
