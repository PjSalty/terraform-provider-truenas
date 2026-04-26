resource "truenas_iscsi_portal" "t" {
  listen {
    ip   = "0.0.0.0"
    port = 3260
  }
}

resource "truenas_iscsi_initiator" "t" {
  comment = "Allow all"
}

resource "truenas_iscsi_target" "example" {
  name  = "iqn.2025-01.com.example:target1"
  alias = "example-target"
  mode  = "ISCSI"

  groups {
    portal    = truenas_iscsi_portal.t.tag
    initiator = tonumber(truenas_iscsi_initiator.t.id)
  }
}
