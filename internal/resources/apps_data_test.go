package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAppsDataSource_basic queries the apps list. Env-gated
// because TrueNAS Apps require a dedicated apps pool configured -
// the test won't pass on an instance where the Apps service isn't
// enabled. The datasource itself doesn't fail when zero apps exist,
// so this validates the Read path round-trips.
func TestAccAppsDataSource_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_APPS") != "1" {
		t.Skip("set TRUENAS_TEST_APPS=1 (and ensure Apps is enabled with an apps pool) to run")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_apps" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_apps.test", "apps.#"),
				),
			},
		},
	})
}
