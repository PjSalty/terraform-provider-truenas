package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSystemDatasetDataSource_basic reads the always-present
// system dataset singleton. No fixture needed — every TrueNAS has
// a system dataset.
func TestAccSystemDatasetDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_systemdataset.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_systemdataset" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "pool"),
					resource.TestCheckResourceAttrSet(dataSourceName, "uuid"),
				),
			},
		},
	})
}
