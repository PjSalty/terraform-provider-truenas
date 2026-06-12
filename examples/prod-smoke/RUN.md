# prod-smoke runbook

This is the runbook for `examples/prod-smoke/`, the workspace you run
against a production TrueNAS the first time you point this provider
at it. It exists to give operators a deterministic, no-side-effects
way to prove the provider can authenticate and read state.

## Three phases

1. **Phase 1: read-only smoke.** Both safety rails armed. The
   provider physically cannot mutate or destroy anything. This is
   what `examples/prod-smoke/main.tf` ships with by default. Run
   this first; expected outcome is "Plan: 0 to add, 0 to change,
   0 to destroy" plus three populated outputs.

2. **Phase 2: apply-safe.** `read_only = false`,
   `destroy_protection = true`. The provider can create and update
   but every DELETE is refused at the wire. This is the right
   configuration for the first apply that ships *new* resources.

3. **Phase 3: full lifecycle.** Both rails dropped. Only run this
   when you have apply-side confidence and a tested backup/restore
   plan. Re-arm `destroy_protection = true` immediately after the
   intentional destroy completes.

The provider attribute names are identical in all three phases -
only the boolean values change, so transitioning between phases is
a one-line diff.

## Phase 1: read-only smoke

### Prerequisites

- TrueNAS SCALE 25.04 or newer (v2.0 ships WebSocket-only and the
  upstream WebSocket endpoint landed in 25.04).
- A TrueNAS API key with at least read access to the pool and
  dataset you'll point at.
- Terraform >= 1.5.
- The provider already published to a registry your `terraform init`
  can reach (registry.terraform.io for public consumers, your
  GitLab provider registry for internal use).

### Setup

```sh
cp -r examples/prod-smoke ~/tf-truenas-prod-smoke
cd ~/tf-truenas-prod-smoke

# Inject the API key from your secret store. Never put it in HCL.
export TF_VAR_truenas_api_key="$(your-secret-store-fetch-command)"

# Point at the target.
export TF_VAR_truenas_url='https://truenas.example.com'
export TF_VAR_smoke_dataset_pool='tank'
export TF_VAR_smoke_dataset_name='path/to/your/existing/dataset'
```

### Run

```sh
terraform init
terraform plan
```

### Expected outcome

```
Changes to Outputs:
  + dataset_id      = "tank/path/to/your/existing/dataset"
  + pool_status     = { ... healthy = true ... }
  + truenas_version = "TrueNAS-SCALE-25.04.x"

Plan: 0 to add, 0 to change, 0 to destroy.
```

### What "good" looks like

- The plan produces zero resource changes.
- All four outputs are populated.
- `truenas_version` matches what the TrueNAS UI shows under
  System Information.
- `pool_status.healthy` is `true`.

### What "bad" looks like, and how to interpret it

| Symptom | Likely cause |
| --- | --- |
| `failed to WebSocket dial: ... 404` on Configure | TrueNAS host is on SCALE 24.x or older. Upgrade SCALE to 25.04+ or stay on the v1.x provider line. |
| `failed to WebSocket dial: ... no such host` | URL typo, DNS misconfiguration, or the firewall rejecting the connection. Curl the URL outside Terraform first. |
| `401 Unauthorized` on the system_info read | API key is wrong, scoped too narrowly, or has been revoked. Generate a fresh one in the TrueNAS UI. |
| `dataset "tank/path" not found` | The named dataset doesn't exist on this pool. Verify with `zfs list -r tank` on the host. |
| Plan shows `~ 0 to change` but with diff lines | A read code path produced different state than the upstream advertises. File an issue with the diff. |


## Phase 2: apply-safe

After Phase 1 is clean, drop `read_only` (creates and updates flow,
delete is still refused at the wire):

```hcl
provider "truenas" {
  url     = var.truenas_url
  api_key = var.truenas_api_key

  read_only          = false  # was true
  destroy_protection = true   # still armed
}
```

You can now extend `main.tf` with new resources. A first apply
that creates ~5 small resources (a dataset, a snapshot_task, a
share, a cronjob, a tunable) is the typical Phase 2 scope.

If your operator pushes a `terraform destroy` by mistake at this
stage, the provider refuses every DELETE call before it reaches
the wire. The TrueNAS host's access log will show no delete
attempts, the safety rail catches them in-process.

## Phase 3: full lifecycle

When you need to delete something intentionally:

```sh
unset TRUENAS_DESTROY_PROTECTION
terraform apply -destroy -target='resource.address'
export TRUENAS_DESTROY_PROTECTION=1   # re-arm immediately
```

Or in HCL, flip `destroy_protection = false` for the duration of
the apply, then flip it back to `true` and re-run plan.

The window during which `destroy_protection = false` is a
deliberately short blast radius, minutes, not hours. The
"re-arm immediately" step is part of the runbook.

## Soak window, when to advance

The plan calls for a 7-day soak between v2.0.0-rc.1 and v2.0.0
during which `terraform plan` runs daily against this workspace
and produces zero drift. The mechanics:

- Run `terraform plan` daily (a CI cron is the easiest setup).
- Any non-empty plan against an unchanged HCL is a regression to
  investigate before tagging v2.0.0.
- Watch for "drift surfaces", a resource attribute that the read
  path computes differently between releases. Schema-stable means
  zero drift on the read path.

## Re-running after a transient failure

The smoke workspace is fully idempotent. Re-running it is safe in
any phase. Specifically:

- Phase 1 cannot persist state changes by design.
- Phase 2's create/update operations are themselves idempotent at
  the provider layer, re-applying a plan that already converged
  is a no-op.
- Phase 3's destroy is one-shot; re-running after a destroy is
  the regular apply lifecycle.
