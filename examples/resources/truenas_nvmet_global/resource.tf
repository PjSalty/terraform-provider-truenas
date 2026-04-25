resource "truenas_nvmet_global" "this" {
  basenqn        = "nqn.2011-06.com.truenas"
  ana            = false
  rdma           = false
  xport_referral = true
}
