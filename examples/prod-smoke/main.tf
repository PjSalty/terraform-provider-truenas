// Copyright (c) PjSalty
// SPDX-License-Identifier: MPL-2.0
//
// prod-smoke: a minimal Terraform workspace for the FIRST run of this
// provider against a production TrueNAS SCALE host. Both safety rails
// are armed by default — `read_only = true` (provider refuses every
// mutating request) plus `destroy_protection = true` (provider refuses
// every DELETE request). With those flags set, this configuration
// physically cannot mutate or destroy anything in the target system,
// so a `terraform apply` here is functionally identical to a
// `terraform plan`.
//
// What this workspace does:
//   1. Reads the system info (a no-op data source, every install has it).
//   2. Imports one existing dataset that you specify by name.
//   3. Asserts the dataset still exists and reports its current state.
//
// What this workspace does NOT do:
//   - Create anything.
//   - Modify anything.
//   - Destroy anything.
//
// Use this as the first contact point against a production system to
// verify the provider can talk to it, authenticate, and read state
// without any risk of side effects. See RUN.md for the full runbook.

terraform {
  required_version = ">= 1.5"
  required_providers {
    truenas = {
      source  = "PjSalty/truenas"
      version = "~> 2.0"
    }
  }
}

provider "truenas" {
  url     = var.truenas_url
  api_key = var.truenas_api_key

  // Safety rails. Both armed. Do NOT clear these on the first run.
  read_only          = true
  destroy_protection = true

  // Tolerate self-signed certs in lab environments. Drop this for prod.
  insecure_skip_verify = var.insecure_skip_verify
}

variable "truenas_url" {
  description = "Base URL of the TrueNAS host (e.g. https://truenas.example.com)."
  type        = string
}

variable "truenas_api_key" {
  description = <<-EOT
    TrueNAS API key. Generate under
    Credentials → Local Users → root → API Keys. Set via
    TF_VAR_truenas_api_key from a secret store; never hardcode.
  EOT
  type        = string
  sensitive   = true
}

variable "smoke_dataset_pool" {
  description = "Name of an existing pool on the target TrueNAS (e.g. \"tank\")."
  type        = string
}

variable "smoke_dataset_name" {
  description = <<-EOT
    Name of an existing dataset under that pool, relative to the pool
    root (e.g. "k8s/postgres"). Must already exist; this workspace will
    NOT create it.
  EOT
  type        = string
}

variable "insecure_skip_verify" {
  description = <<-EOT
    Skip TLS verification on the TrueNAS API connection. Set true
    only for self-signed lab environments.
  EOT
  type        = bool
  default     = false
}

// ---------------------------------------------------------------------
// Read-only verification
// ---------------------------------------------------------------------

data "truenas_system_info" "this" {}

data "truenas_pool" "this" {
  name = var.smoke_dataset_pool
}

data "truenas_dataset" "smoke" {
  id = "${var.smoke_dataset_pool}/${var.smoke_dataset_name}"
}

// ---------------------------------------------------------------------
// Outputs — proof that the provider talked to the box and read state.
// ---------------------------------------------------------------------

output "truenas_version" {
  description = "Version reported by the target TrueNAS host."
  value       = data.truenas_system_info.this.version
}

output "pool_status" {
  description = "Health and free space on the target pool."
  value = {
    name    = data.truenas_pool.this.name
    healthy = data.truenas_pool.this.healthy
    status  = data.truenas_pool.this.status
  }
}

output "dataset_id" {
  description = "Confirmed-existing dataset ID."
  value       = data.truenas_dataset.smoke.id
}

