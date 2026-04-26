data "truenas_certificate" "wildcard" {
  name = "wildcard-example-com"
}

output "cert_id" {
  value = data.truenas_certificate.wildcard.id
}
