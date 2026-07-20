resource "truenas_app" "example" {
  app_name    = "plex"
  catalog_app = "plex"
  train       = "community"
  version     = "1.0.0"
  values = jsonencode({
    plex = {
      claimToken = ""
    }
  })
}

# custom Docker Compose app: exactly one of catalog_app or
# custom_compose is set. Compose edits apply in place; formatting,
# comments, and key order never plan as diffs (comparison is semantic).
resource "truenas_app" "sleeper" {
  app_name       = "sleeper"
  custom_compose = <<-EOT
    services:
      app:
        image: busybox:1.36
        command: ["sleep", "infinity"]
        restart: unless-stopped
  EOT
}
