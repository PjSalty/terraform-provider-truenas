resource "truenas_vm" "example" {
  name             = "examplevm"
  description      = "Example VM managed by Terraform"
  vcpus            = 2
  cores            = 2
  threads          = 1
  memory           = 4096 # MiB
  bootloader       = "UEFI"
  autostart        = true
  time             = "LOCAL"
  shutdown_timeout = 90
}
