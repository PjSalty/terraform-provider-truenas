resource "truenas_dataset" "example" {
  pool        = "tank"
  name        = "example"
  compression = "LZ4"
  atime       = "OFF"
  comments    = "Example dataset managed by Terraform"
}

# Child dataset under an existing parent
resource "truenas_dataset" "child" {
  pool           = "tank"
  parent_dataset = truenas_dataset.example.name
  name           = "child"
  compression    = "LZ4"
}
