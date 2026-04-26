resource "truenas_vm" "example" {
  name   = "examplevm"
  memory = 4096
}

# Disk device backed by a zvol
resource "truenas_zvol" "example_disk" {
  pool    = "tank"
  name    = "vols/examplevm-disk0"
  volsize = 21474836480 # 20 GiB
  sparse  = true
}

resource "truenas_vm_device" "disk" {
  vm    = tonumber(truenas_vm.example.id)
  dtype = "DISK"
  attributes = {
    path = "/dev/zvol/tank/vols/examplevm-disk0"
    type = "VIRTIO"
  }
}

# NIC attached to the bridge
resource "truenas_vm_device" "nic" {
  vm    = tonumber(truenas_vm.example.id)
  dtype = "NIC"
  attributes = {
    type       = "VIRTIO"
    nic_attach = "br0"
  }
}
