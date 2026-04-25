package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccStaticRoute_basic(t *testing.T) {
	resourceName := "truenas_static_route.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckStaticRouteDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccStaticRouteConfigBasic("192.168.99.0/24", "10.10.20.1", "test route"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "destination", "192.168.99.0/24"),
					resource.TestCheckResourceAttr(resourceName, "gateway", "10.10.20.1"),
					resource.TestCheckResourceAttr(resourceName, "description", "test route"),
				),
			},
			// Import by numeric ID
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccStaticRoute_update(t *testing.T) {
	resourceName := "truenas_static_route.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckStaticRouteDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccStaticRouteConfigBasic("192.168.88.0/24", "10.10.20.1", "initial route"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "destination", "192.168.88.0/24"),
					resource.TestCheckResourceAttr(resourceName, "description", "initial route"),
				),
			},
			// Update description and gateway
			{
				Config: testAccStaticRouteConfigBasic("192.168.88.0/24", "10.10.20.1", "updated route"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "updated route"),
				),
			},
		},
	})
}

func testAccCheckStaticRouteDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("static route ID not set")
		}
		return nil
	}
}

func testAccStaticRouteConfigBasic(destination, gateway, description string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_static_route" "test" {
  destination = %q
  gateway     = %q
  description = %q
}
`, destination, gateway, description)
}
