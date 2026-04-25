resource "truenas_iscsi_auth" "example" {
  tag    = 1
  user   = "chap-user"
  secret = "Sup3rSecretCHAP!"
  # peeruser + peersecret are only required for Mutual CHAP
}
