resource "truenas_reporting_exporter" "example" {
  name    = "prometheus"
  type    = "PROMETHEUS"
  enabled = true
  attributes_json = jsonencode({
    port = 9100
  })
}
