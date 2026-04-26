resource "truenas_iscsi_initiator" "example" {
  comment = "Allow trusted networks"
  # Leave initiators unset to allow ALL, or restrict:
  # initiators = ["iqn.2025-01.com.example:client1"]
}
