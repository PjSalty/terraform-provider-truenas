package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestPhaseF_ModifyPlanCoverage exercises the destroy-warning ModifyPlan
// hook on every resource that carries one but doesn't already have a
// dedicated `*_modifyplan_test.go` sibling. Phase F added WarnOnDestroy
// to 14 resources with no prior ModifyPlan logic; those functions need
// at least one test call to satisfy the 100% per-package coverage gate.
//
// The hooks under test are all single-expression:
//
//	func (r *FooResource) ModifyPlan(ctx, req, resp) {
//	    planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_foo")
//	}
//
// so a single null-plan + null-state call per resource hits 100% of the
// function body. The helper WarnOnDestroy itself is already unit-tested
// in internal/planhelpers/destroy_warning_test.go; this test only has
// to ensure each resource-local function body executes at least once.
//
// This is the minimum-viable complement to the static TestPhaseFHooksWired
// ratchet in internal/provider/ — the ratchet verifies the CALL exists
// in source, this test verifies the call actually RUNS under coverage.
func TestPhaseF_ModifyPlanCoverage(t *testing.T) {
	cases := []struct {
		name string
		r    resource.Resource
	}{
		{"cloud_backup", NewCloudBackupResource()},
		{"cloud_sync", NewCloudSyncResource()},
		{"cronjob", NewCronJobResource()},
		{"dataset", NewDatasetResource()},
		{"group", NewGroupResource()},
		{"init_script", NewInitScriptResource()},
		{"iscsi_auth", NewISCSIAuthResource()},
		{"iscsi_portal", NewISCSIPortalResource()},
		{"pool", NewPoolResource()},
		{"rsync_task", NewRsyncTaskResource()},
		{"scrub_task", NewScrubTaskResource()},
		{"share_smb", NewSMBShareResource()},
		{"snapshot_task", NewSnapshotTaskResource()},
		{"user", NewUserResource()},
		{"zvol", NewZvolResource()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mp, ok := tc.r.(resource.ResourceWithModifyPlan)
			if !ok {
				t.Fatalf("%s resource does not implement ResourceWithModifyPlan", tc.name)
			}
			// Null-plan + non-null-state is the canonical destroy shape.
			// callModifyPlanDelete is the shared helper from
			// modifyplan_helpers_test.go that constructs a valid
			// ModifyPlanRequest for the given resource's Schema and
			// feeds it through. It returns the populated response so
			// the test can check diagnostics — here we only assert
			// the call didn't panic and produced no errors.
			resp := callModifyPlanDelete(t, mp)
			if resp.Diagnostics.HasError() {
				t.Errorf("%s.ModifyPlan emitted errors on destroy: %v",
					tc.name, resp.Diagnostics.Errors())
			}
			// The destroy path MUST emit at least one warning (the
			// WarnOnDestroy helper's output). If the helper was
			// accidentally removed from the ModifyPlan body, no
			// warning fires and we catch the regression here — a
			// dynamic complement to the TestDestroyWarningCoverage
			// static ratchet.
			if len(resp.Diagnostics.Warnings()) == 0 {
				t.Errorf("%s.ModifyPlan emitted no warnings on destroy — "+
					"WarnOnDestroy call missing or broken?", tc.name)
			}
		})
	}
}

// TestPhaseE_ConfigValidatorsCoverage exercises the ConfigValidators
// method on every resource that exposes one, verifying the returned
// slice is non-empty and calling Description / MarkdownDescription on
// each validator for sanity. This covers the 3 functions at 0% in the
// internal/resources package (certificate, iscsi_extent,
// network_interface) from the CI coverage gate.
//
// The ConfigValidators functions are all single-expression `return
// []resource.ConfigValidator{ ... }` so a single call per resource
// exercises the full body.
func TestPhaseE_ConfigValidatorsCoverage(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		r    resource.Resource
	}{
		{"certificate", NewCertificateResource()},
		{"iscsi_extent", NewISCSIExtentResource()},
		{"network_interface", NewNetworkInterfaceResource()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cv, ok := tc.r.(resource.ResourceWithConfigValidators)
			if !ok {
				t.Fatalf("%s does not implement ResourceWithConfigValidators", tc.name)
			}
			validators := cv.ConfigValidators(ctx)
			if len(validators) == 0 {
				t.Errorf("%s returned empty ConfigValidators list", tc.name)
			}
			// Touch each validator's Description methods so they count
			// toward function coverage in the resourcevalidators package too.
			for _, v := range validators {
				if d := v.Description(ctx); d == "" {
					t.Errorf("%s validator returned empty Description", tc.name)
				}
				if md := v.MarkdownDescription(ctx); md == "" {
					t.Errorf("%s validator returned empty MarkdownDescription", tc.name)
				}
			}
		})
	}
}
