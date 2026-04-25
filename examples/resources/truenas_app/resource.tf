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
