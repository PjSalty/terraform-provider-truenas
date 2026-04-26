resource "truenas_init_script" "example" {
  type    = "COMMAND"
  when    = "POSTINIT"
  command = "/usr/bin/logger truenas init script executed"
  enabled = true
  comment = "Example init script"
  timeout = 10
}
