# Import a certificate you already hold.
resource "truenas_certificate" "example" {
  name        = "wildcard-example-com"
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = file("${path.module}/cert.pem")
  privatekey  = file("${path.module}/key.pem")
}

# Have TrueNAS generate a signing request. san is required and must list every
# name the certificate is for; a common name on its own is refused. An RSA key
# also needs key_length.
resource "truenas_certificate" "csr" {
  name             = "app-example-com-csr"
  create_type      = "CERTIFICATE_CREATE_CSR"
  common           = "app.example.com"
  san              = ["app.example.com", "www.app.example.com"]
  key_type         = "RSA"
  key_length       = 4096
  digest_algorithm = "SHA256"
  country          = "US"
  state            = "California"
  city             = "San Jose"
  organization     = "Example"
  email            = "hostmaster@example.com"
}

# Satisfy that request through ACME. Every domain in the CSR needs an
# authenticator that can answer its DNS-01 challenge.
resource "truenas_acme_dns_authenticator" "cloudflare" {
  name          = "cloudflare"
  authenticator = "cloudflare"
  attributes = {
    cloudflare_email   = "dns@example.com"
    cloudflare_api_key = var.cloudflare_api_key
  }
}

resource "truenas_certificate" "acme" {
  name               = "app-example-com"
  create_type        = "CERTIFICATE_CREATE_ACME"
  csr_id             = truenas_certificate.csr.id
  acme_directory_uri = "https://acme-v02.api.letsencrypt.org/directory"
  tos                = true
  renew_days         = 10

  dns_mapping = {
    "app.example.com"     = truenas_acme_dns_authenticator.cloudflare.id
    "www.app.example.com" = truenas_acme_dns_authenticator.cloudflare.id
  }
}
