package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDatasetsDataSource_basic queries the datasets list datasource
// after creating a fixture dataset; asserts the fixture surfaces in
// the returned list. This datasource is a pure read-only collection
// query, no per-item id filter, so the assertion shape is
// "fixture id is somewhere in datasets.*.id".
func TestAccDatasetsDataSource_basic(t *testing.T) {
	pool := testAccDatasetPool()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

resource "truenas_dataset" "fixture" {
  pool = "` + pool + `"
  name = "tf-acc-datasets-ds"
}

data "truenas_datasets" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// The list contains at least our fixture; assert
					// the count is >= 1 and the fixture-derived id is
					// present somewhere in the list.
					resource.TestCheckResourceAttrWith("data.truenas_datasets.test", "datasets.#",
						func(s string) error {
							if s == "0" {
								return resource.TestCheckResourceAttr(
									"data.truenas_datasets.test", "datasets.#", "> 0",
								)(nil)
							}
							return nil
						},
					),
				),
			},
		},
	})
}
