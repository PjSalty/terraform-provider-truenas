package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccCatalogDataSource_basic reads the always-present TrueNAS official
// catalog. There is nothing to select by: the catalog is a singleton and its
// id is Computed, so the configuration that used to set id = "TRUENAS" was
// rejected outright with "Cannot set value for this attribute as the provider
// has marked it as read-only". Env-gated because catalog operations touch
// sync metadata that is global to the system.
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

data "truenas_catalog" "test" {}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "TRUENAS"),
					resource.TestCheckResourceAttrSet(dataSourceName, "label"),
				),
			},
		},
	})
}
