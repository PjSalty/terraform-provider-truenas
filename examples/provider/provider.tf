terraform {
  required_providers {
    truenas = {
      source  = "PjSalty/truenas"
      version = "~> 1.0"
    }
  }
}

# Minimal: reads TRUENAS_URL and TRUENAS_API_KEY from the environment.
provider "truenas" {}

# Explicit: supply every argument in HCL. Useful in multi-provider setups
# where the env vars are already bound to a different TrueNAS instance.
# Uncomment and delete the `provider "truenas" {}` line above to use.
#
# provider "truenas" {
#   url     = "https://truenas.example.com"
#   api_key = var.truenas_api_key
# }

# Production safety — Phase 1: read-only plan against prod.
# Every POST/PUT/DELETE is refused at the client layer before it
# reaches the network. The target TrueNAS never even sees the attempt.
#
# provider "truenas" {
#   url             = "https://prod.truenas.example.com"
#   api_key         = var.prod_api_key
#   read_only       = true  # physically unable to mutate anything
#   request_timeout = "5m"  # tuned for loaded prod list endpoints
# }

# Production safety — Phase 3: safe-apply profile. Drops read_only so
# creates and updates flow, but keeps destroy_protection on so no
# resource can be destroyed. This is the recommended first-apply
# configuration against production TrueNAS — equivalent to per-resource
# `deletion_protection` flags found in major Terraform providers, but
# enforced at the wire for every resource in the provider at once.
# See docs/guides/phased-rollout.md.
#
# provider "truenas" {
#   url                = "https://prod.truenas.example.com"
#   api_key            = var.prod_api_key
#   read_only          = false
#   destroy_protection = true  # create + update OK, delete refused
#   request_timeout    = "5m"
# }
