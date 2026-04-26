package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
// When a test environment with pre-seeded container images and
// reliable egress becomes available, this test should be expanded to
// cover full app lifecycle (install, update values, destroy).
func TestAccAppResource_basic(t *testing.T) {
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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_catalog.beta", "id"),
					resource.TestCheckResourceAttr("data.truenas_catalog.beta", "label", "TRUENAS"),
				),
			},
		},
	})
}
