# Singleton — one directory services configuration per system.
resource "truenas_directoryservices" "this" {
  service_type = "ACTIVEDIRECTORY"
  enable       = true
  configuration_json = jsonencode({
    hostname           = "dc1.example.com"
    domainname         = "EXAMPLE.COM"
    bindname           = "administrator"
    bindpw             = "ChangeMe!2026"
    use_default_domain = false
  })
}
