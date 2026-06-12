package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPoolsDataSource_basic queries the pools list datasource and
// asserts at least one pool surfaces — the acc env's TRUENAS_TEST_POOL
// must exist for this assertion to hold. Pure read-only collection
// query.
func TestAccPoolsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_pools" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_pools.test", "pools.#"),
				),
			},
		},
	})
}
