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

func TestAccISCSIAuth_basic(t *testing.T) {
	resourceName := "truenas_iscsi_auth.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIAuthDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIAuthConfigBasic(99, "tf-acc-user", "TfAccSecret12345"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "tag", "99"),
					resource.TestCheckResourceAttr(resourceName, "user", "tf-acc-user"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "peersecret"},
			},
		},
	})
}

func testAccISCSIAuthConfigBasic(tag int, user, secret string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_iscsi_auth" "test" {
  tag    = %d
  user   = %q
  secret = %q
}
`, tag, user, secret)
}

func testAccCheckISCSIAuthExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetISCSIAuth(ctx, id); err != nil {
			return fmt.Errorf("iSCSI auth %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSIAuthDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteISCSIAuth(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of iSCSI auth %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccISCSIAuth_disappears(t *testing.T) {
	resourceName := "truenas_iscsi_auth.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIAuthDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIAuthConfigBasic(98, "tf-acc-disappears", "TfAccDisappear1"),
				Check:  testAccCheckISCSIAuthExists(resourceName),
			},
			{
				Config:             testAccISCSIAuthConfigBasic(98, "tf-acc-disappears", "TfAccDisappear1"),
				Check:              testAccCheckISCSIAuthDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testAccCheckISCSIAuthDestroy verifies the iSCSI auth is gone from
// the upstream after Terraform removes it.
func testAccCheckISCSIAuthDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iscsi_auth ID not set on %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("iscsi_auth ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetISCSIAuth(ctx, id)
		if err == nil {
			return fmt.Errorf("iscsi_auth %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of iscsi_auth %d: %w", id, err)
		}
		return nil
	}
}
