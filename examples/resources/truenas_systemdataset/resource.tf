data "truenas_pool" "tank" {
  name = "tank"
}

resource "truenas_systemdataset" "this" {
  pool = data.truenas_pool.tank.name
}
