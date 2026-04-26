package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_group.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig("wheel"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", "wheel"),
					resource.TestCheckResourceAttrSet(dataSourceName, "gid"),
					resource.TestCheckResourceAttr(dataSourceName, "builtin", "true"),
				),
			},
		},
	})
}

func testAccGroupDataSourceConfig(name string) string {
	return `
provider "truenas" {}

data "truenas_group" "test" {
  name = "` + name + `"
}
`
}
