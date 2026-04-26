resource "truenas_catalog" "example" {
  label            = "custom"
  repository       = "https://github.com/example/catalog.git"
  branch           = "main"
  preferred_trains = ["stable"]
}
