terraform {
  required_providers {
    truenas = {
      source  = "registry.terraform.io/salty/truenas"
      version = "~> 0.1"
    }
  }
}

provider "truenas" {
  url     = "https://truenas.example.com"
  api_key = var.truenas_api_key
}

variable "truenas_api_key" {
  description = "TrueNAS API key"
  type        = string
  sensitive   = true
}
