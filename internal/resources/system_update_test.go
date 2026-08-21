package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSystemUpdate_schemaValidation validates that the
// truenas_system_update resource schema compiles and the HCL config
// parses correctly. It uses the PlanOnly + ExpectNonEmptyPlan pattern
// to confirm the plan computes without actually writing to TrueNAS -
// the destroy_protection + read_only rails in the live smoke test under
// examples/prod-smoke/ handle real prod verification.
//
// This is the required TestAcc function that the acceptanceCoverageFloor
// ratchet (internal/provider/acceptance_coverage_test.go) counts; the
// unit tests for this resource live in system_update_crud_test.go in
// package `resources` so they can access unexported helpers.
func TestAccSystemUpdate_schemaValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping acceptance test in short mode")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testAccSystemUpdateConfigBasic(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// The attributes here are autocheck/profile, not the old auto_download/train.
// The resource was rewritten onto update.config in the 26.0 readiness series
// (issue #32); the previous names were backed by methods that never existed.
//
// This config was never exercised by `go test ./...`, because acceptance
// tests skip without TF_ACC, so the stale HCL survived the rewrite and only
// surfaced when the suite ran against a real 26.0 box.
//
// profile is deliberately omitted: valid profiles are per-system and are
// validated against update.profile_choices at apply time, so hardcoding one
// would fail on any box that does not offer it.
func testAccSystemUpdateConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_system_update" "test" {
  autocheck = false
}
`
}
