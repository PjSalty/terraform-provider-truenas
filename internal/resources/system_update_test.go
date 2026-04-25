package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSystemUpdate_schemaValidation validates that the
// truenas_system_update resource schema compiles and the HCL config
// parses correctly. It uses the PlanOnly + ExpectNonEmptyPlan pattern
// to confirm the plan computes without actually writing to TrueNAS —
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

func testAccSystemUpdateConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_system_update" "test" {
  auto_download = false
  train         = "TrueNAS-SCALE-Fangtooth"
}
`
}
