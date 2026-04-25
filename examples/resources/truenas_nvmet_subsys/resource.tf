resource "truenas_nvmet_subsys" "example" {
  name           = "examplesubsys"
  allow_any_host = false
}
