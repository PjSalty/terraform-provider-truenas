resource "truenas_group" "example" {
  name          = "developers"
  sudo_commands = ["/usr/bin/zfs"]
  smb           = true
}
