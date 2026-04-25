# -----------------------------------------------------------------------------
# Prod TrueNAS smoke test — phased rollout
# -----------------------------------------------------------------------------
#
# PHASE 1 (current): read_only=true
#   Every mutating request (POST/PUT/DELETE) is refused at the CLIENT layer,
#   before any wire call. Provider can refresh state and compute plans, but
#   physically cannot change anything on the target TrueNAS. Run `terraform
#   plan` in this mode to validate drift detection works and no unexpected
#   changes appear against the imported prod state.
#
# PHASE 2: flip read_only=false + destroy_protection=true
#   Creates and updates allowed; DELETE still blocked at the client layer.
#   This is the safe-apply profile: one apply can bring the state file
#   forward but cannot destroy anything.
#
# PHASE 3: brief destroy window
#   Temporarily unset destroy_protection ONLY for the apply that intentionally
#   destroys a resource. Re-arm immediately after.
#
# See docs/guides/phased-rollout.md in the provider repo for the full drill.
# -----------------------------------------------------------------------------

provider "truenas" {
  url     = var.truenas_url
  api_key = var.truenas_api_key

  # -------------------------------------------------------------------------
  # PHASE 1 rail (current): refuse every mutating request at the client
  # layer. This is a physical safety rail — the provider cannot mutate prod
  # even if the HCL has a bug or stray resource block. Flip to false in
  # PHASE 2 after the Phase 1 drift check comes back clean.
  # -------------------------------------------------------------------------
  read_only = true

  # -------------------------------------------------------------------------
  # PHASE 2 rail (dormant while read_only=true): blocks DELETE at the
  # client layer. Meaningful only when read_only=false. Setting it now is
  # belt-and-suspenders — the destroy rail auto-arms when read_only flips
  # but arming it explicitly here means the operator never forgets to set
  # it when they move from Phase 1 to Phase 2.
  # -------------------------------------------------------------------------
  destroy_protection = true

  # -------------------------------------------------------------------------
  # Request timeout: TrueNAS pool/export operations on loaded production
  # systems can legitimately take 60-120s. Raising from the default 60s
  # avoids false-positive timeout errors during first-apply warmup.
  # Value is a Go time.Duration string: "120s", "2m", "1m30s", etc.
  # -------------------------------------------------------------------------
  request_timeout = "120s"
}
