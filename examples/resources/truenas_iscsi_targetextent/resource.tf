resource "truenas_iscsi_targetextent" "example" {
  target = tonumber(truenas_iscsi_target.example.id)
  extent = tonumber(truenas_iscsi_extent.example.id)
  lunid  = 0
}

resource "truenas_iscsi_target" "example" {
  name = "iqn.2025-01.com.example:target1"
  mode = "ISCSI"
  groups {
    portal    = 1
    initiator = 1
  }
}

resource "truenas_iscsi_extent" "example" {
  name = "example-lun"
  type = "DISK"
  disk = "zvol/tank/example"
}
