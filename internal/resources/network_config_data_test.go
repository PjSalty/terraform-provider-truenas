package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNetworkConfigDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_network_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkConfigDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "hostname"),
					resource.TestCheckResourceAttrSet(dataSourceName, "domain"),
				),
			},
		},
	})
}

func testAccNetworkConfigDataSourceConfig() string {
	return `
provider "truenas" {}

data "truenas_network_config" "test" {}
`
}
