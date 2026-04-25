resource "truenas_group" "admins" {
  name = "admins"
}

resource "truenas_user" "example" {
  username          = "alice"
  full_name         = "Alice Example"
  group             = tonumber(truenas_group.admins.id)
  password          = "ChangeMe!2026"
  shell             = "/usr/bin/bash"
  home              = "/var/empty"
  home_mode         = "0755"
  sshpubkey         = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... alice@example"
  password_disabled = false
  locked            = false
}
