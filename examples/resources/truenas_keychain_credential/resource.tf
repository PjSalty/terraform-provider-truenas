resource "truenas_keychain_credential" "key" {
  name = "backup-key"
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = file("~/.ssh/truenas_backup")
    public_key  = file("~/.ssh/truenas_backup.pub")
  }
}

resource "truenas_keychain_credential" "backup" {
  name = "backup-host"
  type = "SSH_CREDENTIALS"
  attributes = {
    host            = "backup.example.com"
    port            = "22"
    username        = "root"
    private_key     = truenas_keychain_credential.key.id
    remote_host_key = "ssh-ed25519 AAAAC3Nza..."
    connect_timeout = "10"
  }
}
