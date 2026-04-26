package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccStaticRouteResource_basic uses the RFC 5737 TEST-NET-1 prefix
// (192.0.2.0/24) so the route is guaranteed not to interfere with
// anything routable on the test network.
func TestAccStaticRouteResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_static_route" "test" {
  destination = "192.0.2.0/24"
  gateway     = "192.0.2.1"
  description = "acctest"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_static_route.test", "destination", "192.0.2.0/24"),
					resource.TestCheckResourceAttr("truenas_static_route.test", "gateway", "192.0.2.1"),
					resource.TestCheckResourceAttrSet("truenas_static_route.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_static_route.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccStaticRouteResource_update verifies that the description can be
// changed in place on an existing static route. Uses 198.51.100.0/24
// (RFC 5737 TEST-NET-2) to avoid collisions with _basic and the disappears
// test, and with any real routes on the test network.
func TestAccStaticRouteResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_static_route" "test" {
  destination = "198.51.100.0/24"
  gateway     = "192.0.2.1"
  description = "initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_static_route.test", "description", "initial"),
					resource.TestCheckResourceAttr("truenas_static_route.test", "destination", "198.51.100.0/24"),
				),
			},
			{
				Config: `
resource "truenas_static_route" "test" {
  destination = "198.51.100.0/24"
  gateway     = "192.0.2.1"
  description = "updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_static_route.test", "description", "updated"),
					resource.TestCheckResourceAttr("truenas_static_route.test", "destination", "198.51.100.0/24"),
				),
			},
		},
	})
}

// TestAccStaticRouteResource_disappears verifies that when a static route
// is deleted out-of-band via the TrueNAS API, the provider detects the
// drift on the next plan. ExpectNonEmptyPlan asserts Terraform produces
// a recreate plan rather than erroring out.
func TestAccStaticRouteResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_static_route" "test" {
  destination = "203.0.113.0/24"
  gateway     = "192.0.2.1"
  description = "acctest-disappears"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_static_route.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_static_route.test"]
						if !ok {
							return fmt.Errorf("truenas_static_route.test not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parse id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteStaticRoute(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
