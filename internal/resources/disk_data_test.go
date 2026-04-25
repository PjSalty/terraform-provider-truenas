package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDiskDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_disk.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDiskDataSourceConfig("sda"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", "sda"),
					resource.TestCheckResourceAttrSet(dataSourceName, "serial"),
					resource.TestCheckResourceAttrSet(dataSourceName, "size"),
					resource.TestCheckResourceAttrSet(dataSourceName, "type"),
					resource.TestCheckResourceAttrSet(dataSourceName, "model"),
				),
			},
		},
	})
}

func testAccDiskDataSourceConfig(name string) string {
	return `
provider "truenas" {}

data "truenas_disk" "test" {
  name = "` + name + `"
}
`
}
