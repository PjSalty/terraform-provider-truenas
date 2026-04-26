resource "truenas_certificate" "example" {
  name        = "wildcard-example-com"
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = file("${path.module}/cert.pem")
  privatekey  = file("${path.module}/key.pem")
}
