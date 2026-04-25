package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNFSConfig_basic(t *testing.T) {
	resourceName := "truenas_nfs_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with defaults
			{
				Config: testAccNFSConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "servers", "2"),
					resource.TestCheckResourceAttr(resourceName, "allow_nonroot", "false"),
					resource.TestCheckResourceAttr(resourceName, "v4_krb", "false"),
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

func TestAccNFSConfig_update(t *testing.T) {
	resourceName := "truenas_nfs_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom servers count
			{
				Config: testAccNFSConfigCustom(4, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "servers", "4"),
					resource.TestCheckResourceAttr(resourceName, "allow_nonroot", "true"),
				),
			},
			// Update servers count
			{
				Config: testAccNFSConfigCustom(2, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "servers", "2"),
					resource.TestCheckResourceAttr(resourceName, "allow_nonroot", "false"),
				),
			},
		},
	})
}

func testAccNFSConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_nfs_config" "test" {}
`
}

func testAccNFSConfigCustom(servers int, allowNonroot bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nfs_config" "test" {
  servers       = %d
  allow_nonroot = %t
}
`, servers, allowNonroot)
}
