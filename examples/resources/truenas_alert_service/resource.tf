resource "truenas_alert_service" "example" {
  name    = "ops-email"
  type    = "Mail"
  enabled = true
  level   = "WARNING"

  settings_json = jsonencode({
    email = "ops@example.com"
  })
}
