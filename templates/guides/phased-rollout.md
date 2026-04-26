---
page_title: "Phased Production Rollout - TrueNAS Provider"
subcategory: "Guides"
description: |-
  How to safely roll the TrueNAS Terraform provider out from a test VM
  to a production TrueNAS SCALE instance without risking data loss.
---

# Phased Production Rollout

This guide is the gate you walk through before pointing
`terraform-provider-truenas` at a production TrueNAS SCALE instance for
the first time. Every step is sequential. Do not skip ahead.

The defensive features this guide relies on were added in v0.5+:

- `read_only` provider attribute and `TRUENAS_READONLY` env var
- `request_timeout` provider attribute and `TRUENAS_REQUEST_TIMEOUT` env var
- `client.IsNotFound` handling on every non-singleton resource's Delete
- Sweeper coverage invariant (no resource can silently go un-swept)
- Apply-idempotency (`ExpectEmptyPlan`) coverage ratchet

If any of the above are missing in your build, stop and upgrade first.

## Rollout phases

The rollout is broken into six phases. Each phase has a
success criterion. Do not advance until the current phase is green.

### Phase 0 — Dry plan against a test VM

Before touching production at all, verify your Terraform config applies
cleanly to a disposable test VM with the **same TrueNAS SCALE version**
as production.

```hcl
provider "truenas" {
  url     = "https://truenas-test.example.com"
  api_key = var.test_api_key
}
```

```sh
terraform init
terraform plan
terraform apply
```

**Success criterion**: `terraform plan` after `apply` reports zero
changes. This is the apply-idempotency gate that v0.5 enforces on
the provider itself; your modules need to meet the same bar.

### Phase 1 — Read-only plan against production

This is the critical phase. Enable the read-only safety rail so the
provider is **physically incapable** of mutating anything, then run
`terraform plan` against production.

```hcl
provider "truenas" {
  url       = "https://truenas.example.com"
  api_key   = var.prod_api_key
  read_only = true
}
```

Or set the env var in your shell:

```sh
export TRUENAS_READONLY=1
terraform plan
```

Expected output:

- Every data source succeeds — reads are allowed.
- Every resource that does NOT yet exist on production shows up as a
  `+ create` line in the plan.
- Every resource that DOES exist on production but is not yet imported
  shows up as a `+ create` line **too** (Terraform does not know about it).
- Any resource that was already imported via `terraform import` shows up
  as an update-to-match or a no-op.

**If the plan shows any unexpected destroys or replacements**, stop. Investigate
why Terraform's model of the resource diverges from TrueNAS's actual state before
proceeding. The read-only rail will have prevented any mutation regardless,
but the divergence itself is a signal that the module is wrong.

**Success criterion**: the plan reflects exactly your intended desired state.
Nothing more, nothing less.

### Phase 2 — Import what already exists

For every resource that production already has and that the plan wants
to `+ create`, run `terraform import` to bring it under management.
Examples:

```sh
terraform import 'truenas_dataset.tank_data' 'tank/data'
terraform import 'truenas_share_nfs.media'   42
terraform import 'truenas_user.jenkins'      1003
```

See [Importing Existing Resources](importing-existing.md) for the
per-resource ID format.

Re-run `terraform plan` (still read-only). Every imported resource
should now show up as no-op. Any updates Terraform wants to apply to
bring the imported resource into line with your HCL should be small
and obviously intentional — if they are not, fix the HCL to match
reality instead of forcing reality to match the HCL.

**Success criterion**: no remaining `+ create` for resources that
actually exist, and every in-place update is clearly intentional.

### Phase 3 — Safe-apply profile: drop read-only, keep destroy-protection

Once the plan looks correct, make the smallest possible change — pick
ONE resource, add a single tag or update a single description field —
and apply it with `read_only = false` but `destroy_protection = true`.
This is the "safe apply" profile: create and update flow through the
wire, but DELETE is still physically refused at the client layer. A
mis-typed HCL removal on any resource hits `ErrDestroyProtected`
instead of destroying production data.

```hcl
provider "truenas" {
  url                = "https://truenas.example.com"
  api_key            = var.prod_api_key
  read_only          = false
  destroy_protection = true  # ← still on — NO resource can be destroyed
}
```

Or via env:

```sh
unset TRUENAS_READONLY
export TRUENAS_DESTROY_PROTECTION=1
```

```sh
terraform plan   # re-verify the diff is the ONE change
terraform apply  # POST/PUT flow; DELETE refused
```

If your HCL accidentally removed a resource, the apply surfaces the
destroy-protected diagnostic instead of destroying anything:

```
Error: Error Deleting Dataset
  truenas client is in destroy-protected mode: refusing to send DELETE request.
  Set destroy_protection=false (or unset TRUENAS_DESTROY_PROTECTION) to allow
  destructive operations: DELETE /pool/dataset/id/tank%2Fimportant
```

That is the expected and correct behavior. Fix the HCL or explicitly
opt into the destroy by dropping the flag, then re-run.

Verify the change landed:

```sh
# Example: confirm the description update via the TrueNAS UI or
# a direct API call. The provider is NOT the source of truth here —
# check reality via a second path.
curl -H "Authorization: Bearer $TRUENAS_API_KEY" \
  https://truenas.example.com/api/v2.0/pool/dataset/id/tank%2Fdata \
  | jq -r .comments
```

**Success criterion**: the change applied cleanly, `terraform plan`
afterward is empty, and out-of-band verification matches your HCL.

### Phase 4 — Tight timeout for the first full apply

Loaded production TrueNAS instances can have slow list endpoints.
If you hit timeouts during Phase 1 or 3, raise the per-request timeout:

```hcl
provider "truenas" {
  url             = "https://truenas.example.com"
  api_key         = var.prod_api_key
  request_timeout = "5m"
}
```

Or:

```sh
export TRUENAS_REQUEST_TIMEOUT=5m
```

A per-request timeout of 2-5 minutes is reasonable for prod. Do not
remove the timeout entirely — values of zero or negative are silently
ignored so the safety rail cannot be disabled.

### Phase 5 — Automate the rollout

Only once Phases 0-4 are green, automate the apply in CI/CD. The CI
job MUST:

1. Run `terraform plan` with `TRUENAS_READONLY=1` first and fail if
   the plan is non-empty or contains any destroy/replace action.
2. Gate the apply step on manual approval (GitHub environment
   protection, GitLab `when: manual`, etc.).
3. Apply with the safety rail OFF only after approval.
4. Re-run `terraform plan` with `TRUENAS_READONLY=1` post-apply and
   fail the pipeline if the plan is non-empty (drift detection).

### Phase 3.5 — Intentional destroy (when you really mean it)

Only clear `destroy_protection` when:

1. The plan shows ≤1 destructive operation AND every destroy is
   intentional and reviewed by a second operator.
2. You have a fresh snapshot of the affected dataset/share/etc.
3. A rollback plan exists (zfs rollback, TrueNAS UI, replication
   target, etc.) and has been verified in the last 24 hours.

Then:

```sh
unset TRUENAS_DESTROY_PROTECTION
terraform plan   # audit one more time
terraform apply  # DELETE now allowed
export TRUENAS_DESTROY_PROTECTION=1  # put the rail back on immediately
```

Re-arm the rail the moment the specific destroy completes. Never
leave a shell session with both rails off.

## Operator emergency brake

If something goes wrong mid-apply:

```sh
# Immediately re-enable BOTH safety rails to prevent any further mutations
export TRUENAS_READONLY=1
export TRUENAS_DESTROY_PROTECTION=1

# Re-run plan to see current state
terraform plan

# If state is inconsistent, target a single resource to recover
terraform plan -target='truenas_dataset.broken'
```

Never `terraform destroy` a production module in a panic. The provider
cannot undo a destroy, and a destroy of a module that includes
`truenas_pool` or `truenas_dataset` resources may delete data you
cannot recover.

## What the provider does NOT protect against

The read-only safety rail prevents the provider from mutating anything.
It does **not** protect against:

- `terraform state rm` — that is a local state edit, not an API call
- `terraform state push` with a crafted state file
- Direct `curl` or `midclt` calls from the host running Terraform
- A human logging into the TrueNAS UI and clicking buttons
- A backup/snapshot policy on TrueNAS destroying data on its own schedule
- Hardware failure, ZFS pool corruption, or a power event

Defense in depth still applies. The provider safety rail is the
last line of defense, not the only one.
