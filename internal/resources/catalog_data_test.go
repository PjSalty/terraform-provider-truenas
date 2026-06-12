package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCatalogDataSource_basic reads the always-present TrueNAS
// official catalog by id. Env-gated because catalog operations are
// expensive (the catalog sync metadata is global to the system).
func TestAccCatalogDataSource_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_CATALOG") != "1" {
		t.Skip("set TRUENAS_TEST_CATALOG=1 to run catalog datasource acc tests")
	}
	dataSourceName := "data.truenas_catalog.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_catalog" "test" {
  id = "TRUENAS"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "TRUENAS"),
					resource.TestCheckResourceAttrSet(dataSourceName, "label"),
				),
			},
		},
	})
}
