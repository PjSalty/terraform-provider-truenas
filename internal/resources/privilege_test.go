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

func TestAccPrivilege_basic(t *testing.T) {
	resourceName := "truenas_privilege.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPrivilegeDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccPrivilegeConfigBasic("tf-acc-priv", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-priv"),
					resource.TestCheckResourceAttr(resourceName, "web_shell", "false"),
					resource.TestCheckResourceAttr(resourceName, "roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "roles.0", "READONLY_ADMIN"),
				),
			},
			// Update: enable web_shell
			{
				Config: testAccPrivilegeConfigBasic("tf-acc-priv", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "web_shell", "true"),
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

func testAccCheckPrivilegeDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("privilege ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("privilege ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetPrivilege(ctx, id)
		if err == nil {
			return fmt.Errorf("privilege %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of privilege %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckPrivilegeExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetPrivilege(ctx, id); err != nil {
			return fmt.Errorf("privilege %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckPrivilegeDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeletePrivilege(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of privilege %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccPrivilege_disappears(t *testing.T) {
	resourceName := "truenas_privilege.test"
	name := fmt.Sprintf("tf-acc-priv-disappears-%d", acctest.ShortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPrivilegeDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccPrivilegeConfigBasic(name, false),
				Check:  testAccCheckPrivilegeExists(resourceName),
			},
			{
				Config:             testAccPrivilegeConfigBasic(name, false),
				Check:              testAccCheckPrivilegeDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccPrivilegeConfigBasic(name string, webShell bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_privilege" "test" {
  name      = %q
  roles     = ["READONLY_ADMIN"]
  web_shell = %t
}
`, name, webShell)
}
