resource "truenas_nvmet_port" "example" {
  addr_trtype  = "TCP"
  addr_traddr  = "10.0.0.20"
  addr_trsvcid = 4420
  addr_adrfam  = "IPV4"
}
