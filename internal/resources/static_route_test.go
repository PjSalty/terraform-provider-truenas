package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
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
				Config: testAccStaticRouteConfigBasic("192.168.99.0/24", "192.0.2.1", "test route"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "destination", "192.168.99.0/24"),
					resource.TestCheckResourceAttr(resourceName, "gateway", "192.0.2.1"),
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
				Config: testAccStaticRouteConfigBasic("192.168.88.0/24", "192.0.2.1", "initial route"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "destination", "192.168.88.0/24"),
					resource.TestCheckResourceAttr(resourceName, "description", "initial route"),
				),
			},
			// Update description and gateway
			{
				Config: testAccStaticRouteConfigBasic("192.168.88.0/24", "192.0.2.1", "updated route"),
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
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("static route ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetStaticRoute(ctx, id)
		if err == nil {
			return fmt.Errorf("static route %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of static route %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckStaticRouteExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetStaticRoute(ctx, id); err != nil {
			return fmt.Errorf("static route %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckStaticRouteDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteStaticRoute(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of static route %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccStaticRoute_disappears(t *testing.T) {
	resourceName := "truenas_static_route.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckStaticRouteDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccStaticRouteConfigBasic("203.0.113.0/24", "192.0.2.1", "tf-acc-disappears"),
				Check:  testAccCheckStaticRouteExists(resourceName),
			},
			{
				Config:             testAccStaticRouteConfigBasic("203.0.113.0/24", "192.0.2.1", "tf-acc-disappears"),
				Check:              testAccCheckStaticRouteDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
