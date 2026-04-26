package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestAccCatalogResource_basic — singleton: TrueNAS only supports one
// catalog (TRUENAS) in SCALE 25.04+. Delete is a no-op (the catalog
// can't be removed), so ImportState+destroy aren't meaningful and the
// test only verifies Create+Read.
func TestAccCatalogResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_catalog" "test" {
  preferred_trains = ["stable"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.#", "1"),
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.0", "stable"),
					resource.TestCheckResourceAttrSet("truenas_catalog.test", "id"),
				),
			},
		},
	})
}

// TestAccCatalogResource_update flips preferred_trains between two
// known-valid values to verify the PUT /catalog path is wired. Only
// `stable` and `community` are universally available across SCALE
// versions so the test sticks to those.
func TestAccCatalogResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_catalog" "test" {
  preferred_trains = ["stable"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.#", "1"),
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.0", "stable"),
				),
			},
			{
				Config: `
resource "truenas_catalog" "test" {
  preferred_trains = ["community", "stable"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.#", "2"),
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.0", "community"),
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.1", "stable"),
				),
			},
		},
	})
}

// TestAccCatalogResource_disappears mutates the catalog preferred_trains
// out-of-band to simulate drift. Because the catalog is a singleton
// that cannot be deleted, a true `disappears` recovery test isn't
// possible; instead we verify that drift in the managed attribute is
// detected by a non-empty plan on the next refresh.
func TestAccCatalogResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_catalog" "test" {
  preferred_trains = ["stable"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_catalog.test", "preferred_trains.0", "stable"),
					func(s *terraform.State) error {
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						drift := []string{"community"}
						_, err = c.UpdateCatalog(ctx, &client.CatalogUpdateRequest{PreferredTrains: &drift})
						return err
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
