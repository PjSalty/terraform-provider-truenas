package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUser_basic(t *testing.T) {
	resourceName := "truenas_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigBasic("tfacctest", "TF Acc Test User"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", "tfacctest"),
					resource.TestCheckResourceAttr(resourceName, "full_name", "TF Acc Test User"),
					resource.TestCheckResourceAttr(resourceName, "locked", "false"),
					resource.TestCheckResourceAttr(resourceName, "smb", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "uid"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Password is write-only and not returned by the API.
				// group_create is only used during creation.
				ImportStateVerifyIgnore: []string{"password", "group_create"},
			},
		},
	})
}

func TestAccUser_update(t *testing.T) {
	resourceName := "truenas_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigBasic("tfaccupdate", "TF Acc Before"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "full_name", "TF Acc Before"),
				),
			},
			// Update full_name in-place
			{
				Config: testAccUserConfigBasic("tfaccupdate", "TF Acc After"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "full_name", "TF Acc After"),
				),
			},
		},
	})
}

func testAccCheckUserDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("user ID not set")
		}
		return nil
	}
}

func testAccUserConfigBasic(username, fullName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_user" "test" {
  username  = %q
  full_name = %q
  password  = "TestP@ssw0rd123!"
}
`, username, fullName)
}
