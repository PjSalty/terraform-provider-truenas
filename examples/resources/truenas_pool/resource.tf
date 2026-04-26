# Creating a pool requires raw disk device names, which is only safe
# on fresh hardware. The topology is passed through as JSON to avoid
# modelling TrueNAS's deeply-nested vdev schema in HCL.
resource "truenas_pool" "example" {
  name = "tank"

  topology_json = jsonencode({
    data = [
      {
        type  = "MIRROR"
        disks = ["sda", "sdb"]
      }
    ]
  })

  encryption = false
}
