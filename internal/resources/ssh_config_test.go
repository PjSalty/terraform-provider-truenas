package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSHConfig_basic(t *testing.T) {
	resourceName := "truenas_ssh_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with defaults
			{
				Config: testAccSSHConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "tcpport", "22"),
					resource.TestCheckResourceAttr(resourceName, "passwordauth", "true"),
					resource.TestCheckResourceAttr(resourceName, "tcpfwd", "false"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateId:           "1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"weak_ciphers"},
			},
		},
	})
}

func TestAccSSHConfig_update(t *testing.T) {
	resourceName := "truenas_ssh_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom port
			{
				Config: testAccSSHConfigCustom(2222, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tcpport", "2222"),
					resource.TestCheckResourceAttr(resourceName, "passwordauth", "false"),
				),
			},
			// Update back to defaults
			{
				Config: testAccSSHConfigCustom(22, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tcpport", "22"),
					resource.TestCheckResourceAttr(resourceName, "passwordauth", "true"),
				),
			},
		},
	})
}

func testAccSSHConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_ssh_config" "test" {}
`
}

func testAccSSHConfigCustom(port int, passwordAuth bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_ssh_config" "test" {
  tcpport      = %d
  passwordauth = %t
}
`, port, passwordAuth)
}
