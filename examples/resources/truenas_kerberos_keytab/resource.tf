resource "truenas_kerberos_keytab" "example" {
  name = "truenas-host"
  # base64-encoded keytab file contents
  file = filebase64("${path.module}/truenas.keytab")
}
