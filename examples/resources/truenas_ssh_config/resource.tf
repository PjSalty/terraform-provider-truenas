resource "truenas_ssh_config" "this" {
  tcpport           = 22
  passwordauth      = false
  kerberosauth      = false
  tcpfwd            = false
  compression       = false
  weak_ciphers      = ["NONE"]
  sftp_log_level    = "ERROR"
  sftp_log_facility = "AUTH"
}
