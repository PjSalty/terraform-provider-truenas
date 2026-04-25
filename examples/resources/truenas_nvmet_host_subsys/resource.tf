resource "truenas_nvmet_host" "h" {
  hostnqn = "nqn.2014-08.org.nvmexpress:uuid:22222222-2222-2222-2222-222222222222"
}

resource "truenas_nvmet_subsys" "s" {
  name = "hostsubsysexample"
}

resource "truenas_nvmet_host_subsys" "example" {
  host_id   = tonumber(truenas_nvmet_host.h.id)
  subsys_id = tonumber(truenas_nvmet_subsys.s.id)
}
