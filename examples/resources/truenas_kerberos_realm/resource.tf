resource "truenas_kerberos_realm" "example" {
  realm          = "EXAMPLE.COM"
  kdc            = ["kdc1.example.com", "kdc2.example.com"]
  admin_server   = ["kdc1.example.com"]
  kpasswd_server = ["kdc1.example.com"]
}
