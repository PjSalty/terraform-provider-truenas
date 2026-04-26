resource "truenas_iscsi_portal" "example" {
  comment = "Primary iSCSI portal"

  listen {
    ip   = "0.0.0.0"
    port = 3260
  }
}
