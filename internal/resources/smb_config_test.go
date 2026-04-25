package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSMBConfig_basic(t *testing.T) {
	resourceName := "truenas_smb_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with defaults
			{
				Config: testAccSMBConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "netbiosname", "truenas"),
					resource.TestCheckResourceAttr(resourceName, "workgroup", "WORKGROUP"),
					resource.TestCheckResourceAttr(resourceName, "enable_smb1", "false"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "1",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSMBConfig_update(t *testing.T) {
	resourceName := "truenas_smb_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom description
			{
				Config: testAccSMBConfigCustom("Test SMB Server", "TESTGROUP"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Test SMB Server"),
					resource.TestCheckResourceAttr(resourceName, "workgroup", "TESTGROUP"),
				),
			},
			// Update description
			{
				Config: testAccSMBConfigCustom("TrueNAS Server", "WORKGROUP"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "TrueNAS Server"),
					resource.TestCheckResourceAttr(resourceName, "workgroup", "WORKGROUP"),
				),
			},
		},
	})
}

func testAccSMBConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_smb_config" "test" {}
`
}

func testAccSMBConfigCustom(description, workgroup string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_smb_config" "test" {
  description = %q
  workgroup   = %q
}
`, description, workgroup)
}
