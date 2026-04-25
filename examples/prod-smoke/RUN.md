# Prod TrueNAS smoke test — runbook

## What this is

A minimal Terraform workspace that imports ONE existing dataset from the
production TrueNAS (10.0.0.10 / truenas.example.com) into state
and verifies the provider can refresh cleanly with zero drift.

This is the Phase 1 safe-apply drill documented in the provider's
`docs/guides/phased-rollout.md` — the goal is to prove the provider can
SEE prod correctly without any ability to mutate anything.

## Prerequisites

1. **Provider installed locally** via `make install` from
   `the provider repo`. This drops the binary into
   `~/.terraform.d/plugins/local/saltstice/truenas/0.1.0/linux_amd64/`
   which `versions.tf` in this workspace resolves against.

2. **SOPS + age keys** set up in `~/.config/sops/age/keys.txt` so
   `sops -d` can decrypt `path/to/secrets.sops.yaml`.

3. **Terraform 1.6+** on PATH.

4. **An existing dataset on prod** to import. Pick something small and
   rarely-changed for the first run (not `tank` itself — pick a child
   dataset).

## Run

```bash
cd ~/tf-truenas-prod-smoke

# 1. Decrypt the prod TrueNAS API key and export it for terraform.
#    The yq pipe strips any surrounding quotes from the SOPS value.
export TF_VAR_truenas_api_key="$(sops -d path/to/secrets.sops.yaml | yq -r '.infrastructure.truenas.api_key' | tr -d '"')"

# 2. Tell terraform which existing dataset to import. MUST already exist.
#    The pool and name-within-pool are separate vars because
#    truenas_dataset models them as distinct required attributes.
export TF_VAR_smoke_dataset_pool="tank"
export TF_VAR_smoke_dataset_name="path/to/your/existing/dataset"

# 3. Sanity-check that the env vars made it through.
echo "url=${TF_VAR_truenas_url:-<default>}  key_len=${#TF_VAR_truenas_api_key}  pool=${TF_VAR_smoke_dataset_pool}  name=${TF_VAR_smoke_dataset_name}"

# 4. Plan. `terraform init` is NOT run because dev_overrides in ~/.terraformrc
#    resolves the provider directly from /tmp/terraform-provider-truenas.
#    Expected output: refresh + "Plan: 0 to add, 0 to change, 0 to destroy."
terraform plan
```

## What you should see

```
Plan: 0 to add, 0 to change, 0 to destroy.
```

If the plan shows ANY changes on the imported dataset, stop and investigate
BEFORE proceeding to Phase 2. The drift means either:
  - the provider's Read path is mapping a field incorrectly, or
  - the server has a setting the provider doesn't know how to represent, or
  - (most likely) the imported dataset has an attribute Terraform's
    plan shows as "drift" because our HCL block doesn't specify it.
    Add the missing attribute to main.tf and re-plan.

## If Phase 1 passes, move to Phase 2

Edit `provider.tf`:
```hcl
provider "truenas" {
  url     = var.truenas_url
  api_key = var.truenas_api_key

  read_only          = false  # <-- flip from true to false
  destroy_protection = true   # <-- keep armed
  request_timeout_seconds = 120
}
```

Re-run `terraform plan`. Should still show zero changes.

Then `terraform apply`. Safe no-op, but proves the non-destructive write
path works end-to-end against prod.

## If any step fails

DO NOT proceed to Phase 3. The emergency brake:

```bash
export TRUENAS_READ_ONLY=1
export TRUENAS_DESTROY_PROTECTION=1
```

Both rails take effect immediately regardless of `provider.tf` — the env
vars override HCL. Then investigate whatever broke before moving forward.

## Phase 3 (intentional destroy window) — NOT part of this workspace

Phase 3 is a one-off: disable `destroy_protection`, apply a single planned
destroy, re-arm immediately. See `docs/guides/phased-rollout.md` in the
provider repo for the full drill. This workspace deliberately doesn't include
a Phase 3 example because a Phase 3 apply against prod should happen
deliberately, not as part of a smoke test.
