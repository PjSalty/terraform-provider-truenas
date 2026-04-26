package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig("root"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "username", "root"),
					resource.TestCheckResourceAttrSet(dataSourceName, "uid"),
					resource.TestCheckResourceAttrSet(dataSourceName, "full_name"),
					resource.TestCheckResourceAttrSet(dataSourceName, "home"),
					resource.TestCheckResourceAttrSet(dataSourceName, "shell"),
					resource.TestCheckResourceAttr(dataSourceName, "builtin", "true"),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig(username string) string {
	return `
provider "truenas" {}

data "truenas_user" "test" {
  username = "` + username + `"
}
`
}
