data "truenas_keychain_credential" "example" {
  id = 1
}

output "keychain_type" {
  value = data.truenas_keychain_credential.example.type
}
