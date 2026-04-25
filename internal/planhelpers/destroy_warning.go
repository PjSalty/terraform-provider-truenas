// Package planhelpers contains reusable resource.ModifyPlan helpers
// that surface operator-facing warnings at plan time. Unlike the
// client-layer destroy-protection rail (internal/client/destroy_protection.go),
// these helpers run inside `terraform plan` — so the operator SEES
// the warning before they run `terraform apply`, not only after a
// wire request is refused. The two layers are complementary:
//
//   - Plan time (this package): warn the operator what WOULD be
//     destroyed when they run apply. Non-blocking.
//   - Apply time (client.DestroyProtection): refuse the wire call
//     outright if destroy_protection=true. Blocking, authoritative.
//
// Together they implement a "see AND brake before the cliff" pattern
// for every destructive resource: the operator cannot miss a planned
// destroy in plan output, and cannot accidentally dispatch it even if
// the HCL change was unintentional.
package planhelpers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// WarnOnDestroy emits a Warning diagnostic to resp.Diagnostics
// whenever the plan is a destroy (plan raw is null and state raw
// is not). The warning message names the resource type and ID so
// operator-reviewed plans cannot miss the destruction visually.
// Call from a resource's ModifyPlan hook:
//
//	func (r *DatasetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
//	    planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_dataset")
//	}
//
// The helper is a no-op for create and update actions — it fires
// only on the specific shape "plan is null, state is not null".
// It cannot be used to BLOCK a destroy (use destroy_protection for
// that) — it only informs.
func WarnOnDestroy(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, resourceType string) {
	// Destroy detection: a destroy plan has a null raw plan and a
	// non-null raw state. Create has null state and non-null plan;
	// update has both non-null.
	if !req.Plan.Raw.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		return
	}

	// The state has an id attribute on every resource in this provider
	// (see TestStatePersistence invariant). Read it as a raw string via
	// the low-level path so this helper stays resource-agnostic. Thread
	// the caller-supplied ctx through so tflog/trace correlation in the
	// framework GetAttribute path is preserved (contextcheck).
	var id string
	// We intentionally ignore the diag here — if the state can't be read,
	// the destroy still warrants a warning, just without the specific ID.
	_ = req.State.GetAttribute(ctx, pathRootID(), &id)
	if id == "" {
		id = "(unknown)"
	}

	resp.Diagnostics.AddWarning(
		fmt.Sprintf("Plan will DESTROY %s %q", resourceType, id),
		fmt.Sprintf(
			"This plan permanently removes %s %q from the TrueNAS instance.\n\n"+
				"Review carefully BEFORE running `terraform apply`. Once applied, "+
				"the resource and any non-snapshotted data it holds may be "+
				"unrecoverable. To block the destroy at the wire (even if the HCL "+
				"change was unintentional), set `destroy_protection = true` on the "+
				"provider block or export TRUENAS_DESTROY_PROTECTION=1. See "+
				"docs/guides/phased-rollout.md Phase 3 for the full drill.",
			resourceType, id,
		),
	)
}
