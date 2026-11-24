# Settings of a disk TrueNAS already knows about. A disk is hardware, so
# this adopts it and never creates or destroys one.
#
# The device name (sda) can change across restarts; the identifier follows
# the drive. Look the identifier up from the name with the data source:
data "truenas_disk" "bay3" {
  name = "sda"
}

resource "truenas_disk" "bay3" {
  identifier = data.truenas_disk.bay3.identifier

  description = "bay 3, archive pool"

  # Spin down after 30 minutes without I/O. Left out, the current value is
  # kept.
  hddstandby = "30"

  # Advanced Power Management: 128 and above never spin the drive down.
  advpowermgmt = "128"
}
