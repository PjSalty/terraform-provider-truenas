// Copyright (c) PjSalty
// SPDX-License-Identifier: MPL-2.0
//
// wsclient-smoke: a tiny Terraform workspace whose only job is to
// prove the WebSocket transport can authenticate and read state from
// the target TrueNAS. Use this when you want to test the WebSocket
// path specifically — for example, when validating that a SCALE
// upgrade did not break the /api/current endpoint, or when
// debugging connectivity through a load balancer.
//
// Differences from prod-smoke:
//   - Does NOT touch any pool or dataset attribute by name.
//     A misconfigured workspace cannot accidentally read out
//     a stale dataset path.
//   - Reads only system_info, which every TrueNAS install
//     populates regardless of pool state.
//
// Use prod-smoke (one directory over) for the full safety-rail
// production drill. Use this for "is the WebSocket transport
// working at all?".

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

  // Read-only: nothing this workspace does can mutate state, but
  // the rail makes that property bit-enforced.
  read_only = true

  insecure_skip_verify = var.insecure_skip_verify
}

variable "truenas_url" {
  description = "Base URL of the TrueNAS host (e.g. https://truenas.example.com)."
  type        = string
}

variable "truenas_api_key" {
  description = "TrueNAS API key. Inject via TF_VAR_truenas_api_key from a secret store."
  type        = string
  sensitive   = true
}

variable "insecure_skip_verify" {
  description = "Skip TLS verification for self-signed lab environments."
  type        = bool
  default     = false
}

data "truenas_system_info" "this" {}

output "truenas_version" {
  description = "Version reported by the target TrueNAS host."
  value       = data.truenas_system_info.this.version
}

output "transport_verified" {
  description = "Sentinel that proves Configure ran the WebSocket branch."
  value       = "websocket"
}
