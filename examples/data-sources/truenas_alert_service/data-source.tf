data "truenas_alert_service" "example" {
  id = 1
}

output "alert_service_type" {
  value = data.truenas_alert_service.example.type
}
