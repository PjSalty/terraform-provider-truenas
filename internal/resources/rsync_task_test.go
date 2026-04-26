package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRsyncTask_schemaValidation(t *testing.T) {
	// This test validates that the rsync task resource schema compiles and
	// the configuration parses correctly. Actual rsync task creation requires
	// a remote host to be configured.
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
				// without actually creating the resource (no remote host).
				Config:             testAccRsyncTaskConfigBasic(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRsyncTaskConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_rsync_task" "test" {
  path         = "/mnt/tank/data"
  user         = "root"
  remotehost   = "192.168.1.100"
  remotemodule = "backup"
  mode         = "MODULE"
  direction    = "PUSH"
  enabled      = false
  desc         = "Terraform acceptance test"
}
`
}
