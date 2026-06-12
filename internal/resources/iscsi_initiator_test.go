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

func TestAccISCSIInitiator_basic(t *testing.T) {
	resourceName := "truenas_iscsi_initiator.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIInitiatorDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIInitiatorConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "comment", "tf-acc-test initiator"),
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

func testAccCheckISCSIInitiatorDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI initiator ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("iSCSI initiator ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetISCSIInitiator(ctx, id)
		if err == nil {
			return fmt.Errorf("iSCSI initiator %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of iSCSI initiator %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSIInitiatorExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetISCSIInitiator(ctx, id); err != nil {
			return fmt.Errorf("iSCSI initiator %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSIInitiatorDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteISCSIInitiator(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of iSCSI initiator %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccISCSIInitiator_disappears(t *testing.T) {
	resourceName := "truenas_iscsi_initiator.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIInitiatorDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIInitiatorConfigBasic(),
				Check:  testAccCheckISCSIInitiatorExists(resourceName),
			},
			{
				Config:             testAccISCSIInitiatorConfigBasic(),
				Check:              testAccCheckISCSIInitiatorDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccISCSIInitiatorConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_iscsi_initiator" "test" {
  comment = "tf-acc-test initiator"
}
`
}
