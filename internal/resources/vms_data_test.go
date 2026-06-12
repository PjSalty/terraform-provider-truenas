package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccVMsDataSource_basic queries the VMs list. No fixture
// needed, the list datasource returns whatever VMs exist (zero
// is a valid result on a fresh test instance). Validates the
// datasource Read path itself, not specific VM data.
func TestAccVMsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_vms" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_vms.test", "vms.#"),
				),
			},
		},
	})
}
