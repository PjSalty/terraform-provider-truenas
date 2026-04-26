package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDirectoryServices_disable verifies the singleton resource can be
// created in a disabled state. We avoid actually joining an AD/LDAP/IPA
// domain in acceptance tests because it requires live directory servers.
func TestAccDirectoryServices_disable(t *testing.T) {
	resourceName := "truenas_directoryservices.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryServicesDisabled(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "enable", "false"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateId:           "1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential_json", "configuration_json", "force"},
			},
		},
	})
}

func testAccDirectoryServicesDisabled() string {
	return `
provider "truenas" {}

resource "truenas_directoryservices" "test" {
  enable = false
}
`
}
