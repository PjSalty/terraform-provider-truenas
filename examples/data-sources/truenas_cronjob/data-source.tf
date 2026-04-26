data "truenas_cronjob" "example" {
  id = 1
}

output "cronjob_command" {
  value = data.truenas_cronjob.example.command
}
