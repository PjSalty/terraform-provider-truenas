resource "truenas_zvol" "nvme_ns" {
  pool    = "tank"
  name    = "vols/nvme-ns0"
  volsize = 10737418240
  sparse  = true
}

resource "truenas_nvmet_subsys" "ns" {
  name = "nsexample"
}

resource "truenas_nvmet_namespace" "example" {
  subsys_id   = tonumber(truenas_nvmet_subsys.ns.id)
  device_type = "ZVOL"
  device_path = "zvol/tank/vols/nvme-ns0"
  enabled     = true
}
