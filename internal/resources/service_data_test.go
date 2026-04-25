package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceDataSourceConfig("ssh"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "service", "ssh"),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "state"),
				),
			},
		},
	})
}

func testAccServiceDataSourceConfig(service string) string {
	return `
provider "truenas" {}

data "truenas_service" "test" {
  service = "` + service + `"
}
`
}
