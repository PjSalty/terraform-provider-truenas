package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCloudSync_schemaValidation(t *testing.T) {
	// This test validates that the cloud sync resource schema compiles and
	// the configuration parses correctly. Actual cloud sync creation requires
	// cloud credentials to be configured on the TrueNAS instance.
	if testing.Short() {
		t.Skip("Skipping acceptance test in short mode")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Validates the schema and config parsing work correctly.
				// PlanOnly + ExpectNonEmptyPlan confirms the plan computes
				// without actually creating the resource (no cloud credentials).
				Config:             testAccCloudSyncConfigBasic(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCloudSyncConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_cloud_sync" "test" {
  path          = "/mnt/tank/backup"
  credentials   = 1
  direction     = "PUSH"
  transfer_mode = "SYNC"
  description   = "Terraform acceptance test"
  enabled       = false
}
`
}
