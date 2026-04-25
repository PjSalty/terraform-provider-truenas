package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCertificateDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_certificate.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateDataSourceConfig("truenas_default"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", "truenas_default"),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "valid_from"),
					resource.TestCheckResourceAttrSet(dataSourceName, "valid_until"),
					resource.TestCheckResourceAttr(dataSourceName, "expired", "false"),
				),
			},
		},
	})
}

func testAccCertificateDataSourceConfig(name string) string {
	return `
provider "truenas" {}

data "truenas_certificate" "test" {
  name = "` + name + `"
}
`
}
