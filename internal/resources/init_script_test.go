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

func TestAccInitScript_basic(t *testing.T) {
	resourceName := "truenas_init_script.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInitScriptDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccInitScriptConfigBasic("echo hello", "POSTINIT"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "type", "COMMAND"),
					resource.TestCheckResourceAttr(resourceName, "command", "echo hello"),
					resource.TestCheckResourceAttr(resourceName, "when", "POSTINIT"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "timeout", "10"),
				),
			},
			// Update
			{
				Config: testAccInitScriptConfigBasic("echo world", "PREINIT"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "command", "echo world"),
					resource.TestCheckResourceAttr(resourceName, "when", "PREINIT"),
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

func testAccCheckInitScriptDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("init script ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("init script ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetInitScript(ctx, id)
		if err == nil {
			return fmt.Errorf("init script %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of init script %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckInitScriptExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetInitScript(ctx, id); err != nil {
			return fmt.Errorf("init script %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckInitScriptDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteInitScript(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of init script %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccInitScript_disappears(t *testing.T) {
	resourceName := "truenas_init_script.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInitScriptDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccInitScriptConfigBasic("echo disappears-test", "POSTINIT"),
				Check:  testAccCheckInitScriptExists(resourceName),
			},
			{
				Config:             testAccInitScriptConfigBasic("echo disappears-test", "POSTINIT"),
				Check:              testAccCheckInitScriptDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccInitScriptConfigBasic(command, when string) string {
	return `
provider "truenas" {}

resource "truenas_init_script" "test" {
  type    = "COMMAND"
  command = "` + command + `"
  when    = "` + when + `"
  enabled = true
  timeout = 10
}
`
}
