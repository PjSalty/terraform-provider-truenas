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

func TestAccAlertService_basic(t *testing.T) {
	resourceName := "truenas_alert_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAlertServiceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertServiceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-alert"),
					resource.TestCheckResourceAttr(resourceName, "type", "Mail"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "level", "WARNING"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"settings_json"},
			},
		},
	})
}

func testAccCheckAlertServiceDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("alert service ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("alert service ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetAlertService(ctx, id)
		if err == nil {
			return fmt.Errorf("alert service %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of alert service %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckAlertServiceExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetAlertService(ctx, id); err != nil {
			return fmt.Errorf("alert service %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckAlertServiceDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteAlertService(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of alert service %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccAlertService_disappears(t *testing.T) {
	resourceName := "truenas_alert_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAlertServiceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertServiceConfigBasic(),
				Check:  testAccCheckAlertServiceExists(resourceName),
			},
			{
				Config:             testAccAlertServiceConfigBasic(),
				Check:              testAccCheckAlertServiceDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAlertServiceConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_alert_service" "test" {
  name  = "tf-acc-test-alert"
  type  = "Mail"
  level = "WARNING"

  settings_json = jsonencode({
    email = "tf-acc-test@example.com"
  })
}
`
}
