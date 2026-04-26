resource "truenas_nvmet_port" "p" {
  addr_trtype  = "TCP"
  addr_traddr  = "10.0.0.20"
  addr_trsvcid = 4420
}

resource "truenas_nvmet_subsys" "s2" {
  name = "portsubsysexample"
}

resource "truenas_nvmet_port_subsys" "example" {
  port_id   = tonumber(truenas_nvmet_port.p.id)
  subsys_id = tonumber(truenas_nvmet_subsys.s2.id)
}
