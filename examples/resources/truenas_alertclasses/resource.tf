# Singleton: global alert-class policy overrides.
resource "truenas_alertclasses" "this" {
  classes = {
    ZpoolCapacityWarning = {
      level  = "WARNING"
      policy = "IMMEDIATELY"
    }
    ZpoolCapacityCritical = {
      level  = "CRITICAL"
      policy = "IMMEDIATELY"
    }
  }
}
