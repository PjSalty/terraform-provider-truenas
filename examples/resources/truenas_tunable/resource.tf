resource "truenas_tunable" "example" {
  type    = "SYSCTL"
  var     = "vfs.zfs.arc_max"
  value   = "17179869184" # 16 GiB
  comment = "Cap ARC at 16 GiB"
  enabled = true
}
