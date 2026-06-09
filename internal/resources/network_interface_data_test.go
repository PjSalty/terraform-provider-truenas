package resources_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNetworkInterfaceDataSource_basic queries an existing network
// interface by id. We don't create a fixture because creating network
// interfaces is invasive (may require a known physical parent NIC,
// can disconnect the test TrueNAS from the lab network) — we
// env-gate the test on TRUENAS_TEST_PARENT_INTERFACE and read the
// pre-existing primary interface.
func TestAccNetworkInterfaceDataSource_basic(t *testing.T) {
	parent := os.Getenv("TRUENAS_TEST_PARENT_INTERFACE")
	if parent == "" {
		t.Skip("network_interface ds test needs TRUENAS_TEST_PARENT_INTERFACE pointing at a known interface name (eg. eno1, ens18)")
	}
	dataSourceName := "data.truenas_network_interface.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_network_interface" "test" {
  id = %q
}
`, parent),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", parent),
					resource.TestCheckResourceAttr(dataSourceName, "name", parent),
				),
			},
		},
	})
}

func TestAccNetworkInterfaceDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_network_interface" "test" {
  id = "definitely-not-a-real-interface-99"
}
`,
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
