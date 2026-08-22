resource "truenas_zvol" "example" {
  pool         = "tank"
  name         = "vols/example"
  volsize      = 10737418240 # 10 GiB
  volblocksize = "16K"
  compression  = "LZ4"
  comments     = "Example zvol for iSCSI"
}
