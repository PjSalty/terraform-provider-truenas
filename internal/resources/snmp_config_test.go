package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSNMPConfig_basic(t *testing.T) {
	resourceName := "truenas_snmp_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with defaults
			{
				Config: testAccSNMPConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "community", "public"),
					resource.TestCheckResourceAttr(resourceName, "v3", "false"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateId:           "1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"v3_password", "v3_privpassphrase"},
			},
		},
	})
}

func TestAccSNMPConfig_update(t *testing.T) {
	resourceName := "truenas_snmp_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom community
			{
				Config: testAccSNMPConfigCustom("private", "Server Room"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "community", "private"),
					resource.TestCheckResourceAttr(resourceName, "location", "Server Room"),
				),
			},
			// Update community
			{
				Config: testAccSNMPConfigCustom("public", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "community", "public"),
					resource.TestCheckResourceAttr(resourceName, "location", ""),
				),
			},
		},
	})
}

func testAccSNMPConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_snmp_config" "test" {}
`
}

func testAccSNMPConfigCustom(community, location string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_snmp_config" "test" {
  community = %q
  location  = %q
}
`, community, location)
}
