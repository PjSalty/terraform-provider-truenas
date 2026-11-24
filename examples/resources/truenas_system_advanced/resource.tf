# System > Advanced settings. This is a singleton: TrueNAS has exactly one.
# Every argument is optional, and one left out keeps its current value.

resource "truenas_system_advanced" "this" {
  # Ship logs to up to two remote syslog servers. The list is ordered and
  # replaces whatever is stored; an empty list removes every server.
  syslogservers = [
    { host = "192.0.2.10" },                               # UDP, port 514
    { host = "logs.example.com:6514", transport = "TLS" }, # one-way TLS
  ]
  sysloglevel  = "F_INFO"
  syslog_audit = true # also send audit messages to the servers

  motd         = "Managed by Terraform"
  login_banner = "Authorised use only."

  # Days between boot pool scrubs.
  boot_scrub = 7
}
