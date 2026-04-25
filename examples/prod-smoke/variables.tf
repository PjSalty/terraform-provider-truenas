variable "truenas_url" {
  description = "Base URL of the prod TrueNAS SCALE instance. Must be HTTPS."
  type        = string
  default     = "https://truenas.example.com"

  validation {
    condition     = startswith(var.truenas_url, "https://")
    error_message = "truenas_url must start with https:// — the provider refuses to ship api keys over plaintext."
  }
}

variable "truenas_api_key" {
  description = "TrueNAS API key in <id>-<secret> form. Set via TF_VAR_truenas_api_key, NEVER hardcoded."
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.truenas_api_key) > 20
    error_message = "truenas_api_key looks too short — did you export TF_VAR_truenas_api_key correctly?"
  }
}

variable "smoke_dataset_pool" {
  description = "Pool name of the existing dataset to import (e.g. 'tank')."
  type        = string
}

variable "smoke_dataset_name" {
  description = <<-EOT
    Dataset name relative to the pool of the existing dataset the smoke test
    will IMPORT and refresh. MUST exist before `terraform plan` runs (this
    workspace never creates it). E.g. if smoke_dataset_pool = "tank" and
    the actual path is tank/k8s/example, set this to "k8s/example".
    Pick a small, rarely-touched dataset so any drift is easy to spot.
  EOT
  type        = string
}
