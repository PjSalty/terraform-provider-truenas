package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccAppResource_basic is a **BETA** acceptance test.
//
// Installing a real TrueNAS SCALE application takes 2-5 minutes, may
// require network access to container registries, and has a high flake
// rate against fresh test VMs. Rather than install an app, this test
// verifies that the singleton TRUENAS catalog is present (SCALE ships
// it pre-installed) via the truenas_catalog data source. This gives
// basic coverage that the provider's app/catalog wiring is reachable
// without the cost of a full install.
//
// It is deliberately NOT named _basic. It reads the catalog data source and
// never touches truenas_app, so calling it the app resource's basic test would
// credit the resource with lifecycle coverage it does not have, which is what
// the lifecycle invariant was reading it as. App's real coverage is
// TestAccApp_basic in internal/resources/app_test.go, gated on
// TRUENAS_TEST_APPS because installing an app needs pre-seeded container
// images and reliable egress.
//
// What this does check is worth keeping: that the catalog an app install would
// resolve against is reachable and labelled as expected.
func TestAccAppResource_catalogReachable(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "truenas_catalog" "beta" {}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_catalog.beta", "id"),
					resource.TestCheckResourceAttr("data.truenas_catalog.beta", "label", "TRUENAS"),
				),
			},
		},
	})
}
