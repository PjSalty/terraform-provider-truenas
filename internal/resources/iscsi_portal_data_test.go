package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIPortalDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_iscsi_portal.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

resource "truenas_iscsi_portal" "fixture" {
  comment = "tf-acc-portal-ds"

  listen = [{
    ip = "0.0.0.0"
  }]
}

data "truenas_iscsi_portal" "test" {
  id = truenas_iscsi_portal.fixture.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "comment", "tf-acc-portal-ds"),
				),
			},
		},
	})
}

func TestAccISCSIPortalDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_iscsi_portal" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
