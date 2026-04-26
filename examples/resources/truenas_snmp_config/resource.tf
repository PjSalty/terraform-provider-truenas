resource "truenas_snmp_config" "this" {
  location  = "Rack 1"
  contact   = "ops@example.com"
  community = "public"
  v3        = false
  traps     = false
  zilstat   = false
  loglevel  = 3
}
