resource "truenas_group" "example" {
  name                   = "developers"
  sudo_commands_nopasswd = ["/usr/bin/zfs"]
  smb                    = true
}
