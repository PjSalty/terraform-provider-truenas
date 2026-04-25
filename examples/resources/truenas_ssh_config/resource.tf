resource "truenas_ssh_config" "this" {
  tcpport           = 22
  rootlogin         = false
  passwordauth      = false
  kerberosauth      = false
  tcpfwd            = false
  compression       = false
  sftp_log_level    = "ERROR"
  sftp_log_facility = "AUTH"
}
