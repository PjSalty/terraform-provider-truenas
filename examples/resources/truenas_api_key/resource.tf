resource "truenas_user" "svc" {
  username  = "svcterraform"
  full_name = "Terraform Service Account"
  group     = 0
  password  = "rotated-by-terraform"
}

resource "truenas_api_key" "example" {
  name     = "terraform-ci"
  username = truenas_user.svc.username
}
