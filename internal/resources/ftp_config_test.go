package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFTPConfig_basic(t *testing.T) {
	resourceName := "truenas_ftp_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with defaults
			{
				Config: testAccFTPConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "port", "21"),
					resource.TestCheckResourceAttr(resourceName, "clients", "5"),
					resource.TestCheckResourceAttr(resourceName, "defaultroot", "true"),
					resource.TestCheckResourceAttr(resourceName, "tls", "false"),
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

func TestAccFTPConfig_update(t *testing.T) {
	resourceName := "truenas_ftp_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom settings
			{
				Config: testAccFTPConfigCustom(2121, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "port", "2121"),
					resource.TestCheckResourceAttr(resourceName, "clients", "10"),
				),
			},
			// Update settings
			{
				Config: testAccFTPConfigCustom(21, 5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "port", "21"),
					resource.TestCheckResourceAttr(resourceName, "clients", "5"),
				),
			},
		},
	})
}

func testAccFTPConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_ftp_config" "test" {}
`
}

func testAccFTPConfigCustom(port, clients int) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_ftp_config" "test" {
  port    = %d
  clients = %d
}
`, port, clients)
}
