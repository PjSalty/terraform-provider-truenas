resource "truenas_acme_dns_authenticator" "example" {
  name          = "cloudflare"
  authenticator = "cloudflare"
  attributes = {
    cloudflare_email   = "dns@example.com"
    cloudflare_api_key = "changeme"
  }
}
