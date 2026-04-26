package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		return nil
	}
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
