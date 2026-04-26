data "truenas_api_key" "example" {
  id = 1
}

output "api_key_revoked" {
  value = data.truenas_api_key.example.revoked
}
