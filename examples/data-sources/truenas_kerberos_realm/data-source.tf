data "truenas_kerberos_realm" "example" {
  id = 1
}

output "realm" {
  value = data.truenas_kerberos_realm.example.realm
}
