resource "truenas_reporting_exporter" "example" {
  name    = "graphite"
  enabled = true

  # The exporter type is part of the attribute payload, not a separate field.
  attributes_json = jsonencode({
    exporter_type    = "GRAPHITE"
    destination_ip   = "10.0.0.50"
    destination_port = 2003
    namespace        = "truenas"
  })
}
