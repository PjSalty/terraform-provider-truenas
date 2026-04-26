data "truenas_app" "plex" {
  id = "plex"
}

output "app_state" {
  value = data.truenas_app.plex.state
}
