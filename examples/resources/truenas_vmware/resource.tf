resource "truenas_vmware" "example" {
  hostname   = "vcenter.example.com"
  username   = "administrator@vsphere.local"
  password   = "ChangeMe!2026"
  datastore  = "truenas-nfs"
  filesystem = "tank/vmware"
}
