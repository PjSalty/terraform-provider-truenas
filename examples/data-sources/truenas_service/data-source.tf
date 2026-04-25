data "truenas_service" "ssh" {
  service = "ssh"
}

output "ssh_state" {
  value = data.truenas_service.ssh.state
}
