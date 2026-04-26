package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		return nil
	}
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
