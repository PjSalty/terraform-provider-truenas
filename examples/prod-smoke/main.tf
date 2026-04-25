# -----------------------------------------------------------------------------
# PHASE 1 smoke test: import one existing dataset, verify zero drift.
# -----------------------------------------------------------------------------
#
# This file imports an existing prod dataset into Terraform state and declares
# a matching resource block. On `terraform plan`:
#   - If the imported state matches the resource block: zero changes. PASS.
#   - If anything drifts: plan shows the diff. INVESTIGATE before proceeding.
#
# No `terraform apply` is possible in Phase 1 because read_only=true refuses
# every mutating request at the client layer. This is by design — the goal of
# Phase 1 is to prove the provider can SEE prod state correctly without any
# ability to change it.
#
# To proceed to Phase 2 after Phase 1 goes clean:
#   1. In provider.tf, flip `read_only = false`
#      (leave `destroy_protection = true` armed)
#   2. Run `terraform plan` again — should still show zero changes
#   3. Run `terraform apply` — safe no-op that proves auth + non-destructive
#      wire path works end-to-end against prod
# -----------------------------------------------------------------------------

# Import the existing dataset into state. The composed id is the full
# dataset path that TrueNAS reports (e.g. "tank/k8s/example"). Both the
# pool and the name-within-pool must be provided separately because the
# truenas_dataset schema models them as distinct required attributes.
# This block never CREATES — the dataset must already exist on prod.
# On first run, terraform plan walks the import block, fetches the
# current server state via the Read path, and places it in local state.
import {
  to = truenas_dataset.smoke
  id = "${var.smoke_dataset_pool}/${var.smoke_dataset_name}"
}

resource "truenas_dataset" "smoke" {
  pool = var.smoke_dataset_pool
  name = var.smoke_dataset_name

  # No other attributes specified — the provider will populate them from
  # the server during the import read. This tests the canonical "import
  # an existing resource and refresh state" flow, which is the #1 workflow
  # for migrating an existing TrueNAS into Terraform management.
}

# -----------------------------------------------------------------------------
# OUTPUTS — what we learned about the dataset after import.
# These are read-only; evaluating them confirms the Read path works end-to-end.
# -----------------------------------------------------------------------------

output "smoke_dataset_id" {
  description = "Resolved dataset ID from the import."
  value       = truenas_dataset.smoke.id
}

output "smoke_dataset_full_name" {
  description = "Full pool/name path after import."
  value       = "${truenas_dataset.smoke.pool}/${truenas_dataset.smoke.name}"
}
