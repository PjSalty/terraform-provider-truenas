terraform {
  required_version = ">= 1.6.0"

  required_providers {
    truenas = {
      # Matches the dev_override in ~/.terraformrc which points
      # registry.terraform.io/saltstice/truenas at /tmp (where the binary
      # is staged by `cp bin/terraform-provider-truenas /tmp/` — see RUN.md).
      # `terraform init` is NOT required under dev_overrides; `terraform
      # plan` resolves the provider directly from the override path.
      source = "saltstice/truenas"
    }
  }
}
