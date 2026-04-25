package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNetworkConfig_basic(t *testing.T) {
	resourceName := "truenas_network_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with defaults
			{
				Config: testAccNetworkConfigResourceBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "hostname"),
					resource.TestCheckResourceAttrSet(resourceName, "domain"),
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

func TestAccNetworkConfig_update(t *testing.T) {
	resourceName := "truenas_network_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom domain
			{
				Config: testAccNetworkConfigResourceCustom("truenas", "test.local"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hostname", "truenas"),
					resource.TestCheckResourceAttr(resourceName, "domain", "test.local"),
				),
			},
			// Update domain
			{
				Config: testAccNetworkConfigResourceCustom("truenas", "local"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hostname", "truenas"),
					resource.TestCheckResourceAttr(resourceName, "domain", "local"),
				),
			},
		},
	})
}

func testAccNetworkConfigResourceBasic() string {
	return `
provider "truenas" {}

resource "truenas_network_config" "test" {}
`
}

func testAccNetworkConfigResourceCustom(hostname, domain string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_network_config" "test" {
  hostname = %q
  domain   = %q
}
`, hostname, domain)
}
