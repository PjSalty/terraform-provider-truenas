package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccISCSIPortal_basic(t *testing.T) {
	resourceName := "truenas_iscsi_portal.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIPortalDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIPortalConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "comment", "tf-acc-test portal"),
					resource.TestCheckResourceAttrSet(resourceName, "tag"),
					resource.TestCheckResourceAttr(resourceName, "listen.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "listen.0.ip", "0.0.0.0"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckISCSIPortalDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI portal ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("iSCSI portal ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetISCSIPortal(ctx, id)
		if err == nil {
			return fmt.Errorf("iSCSI portal %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of iSCSI portal %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSIPortalExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetISCSIPortal(ctx, id); err != nil {
			return fmt.Errorf("iSCSI portal %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSIPortalDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteISCSIPortal(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of iSCSI portal %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccISCSIPortal_disappears(t *testing.T) {
	resourceName := "truenas_iscsi_portal.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIPortalDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIPortalConfigBasic(),
				Check:  testAccCheckISCSIPortalExists(resourceName),
			},
			{
				Config:             testAccISCSIPortalConfigBasic(),
				Check:              testAccCheckISCSIPortalDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccISCSIPortalConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_iscsi_portal" "test" {
  comment = "tf-acc-test portal"

  listen = [{
    ip = "0.0.0.0"
  }]
}
`
}
