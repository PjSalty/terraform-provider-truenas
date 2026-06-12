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

func TestAccISCSITarget_basic(t *testing.T) {
	resourceName := "truenas_iscsi_target.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSITargetDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-target"),
					resource.TestCheckResourceAttr(resourceName, "mode", "ISCSI"),
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

func testAccCheckISCSITargetDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI target ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("iSCSI target ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetISCSITarget(ctx, id)
		if err == nil {
			return fmt.Errorf("iSCSI target %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of iSCSI target %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSITargetExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetISCSITarget(ctx, id); err != nil {
			return fmt.Errorf("iSCSI target %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSITargetDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteISCSITarget(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of iSCSI target %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccISCSITarget_disappears(t *testing.T) {
	resourceName := "truenas_iscsi_target.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSITargetDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetConfigBasic(),
				Check:  testAccCheckISCSITargetExists(resourceName),
			},
			{
				Config:             testAccISCSITargetConfigBasic(),
				Check:              testAccCheckISCSITargetDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccISCSITargetConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_iscsi_portal" "dep" {
  comment = "tf-acc-target-dep portal"

  listen = [{
    ip = "0.0.0.0"
  }]
}

resource "truenas_iscsi_initiator" "dep" {
  comment = "tf-acc-target-dep initiator"
}

resource "truenas_iscsi_target" "test" {
  name = "tf-acc-test-target"

  groups = [{
    portal    = truenas_iscsi_portal.dep.id
    initiator = truenas_iscsi_initiator.dep.id
  }]
}
`
}
