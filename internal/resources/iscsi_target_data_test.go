package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSITargetDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_iscsi_target.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

resource "truenas_iscsi_target" "fixture" {
  name = "tf-acc-target-ds"
  mode = "ISCSI"
}

data "truenas_iscsi_target" "test" {
  id = truenas_iscsi_target.fixture.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-acc-target-ds"),
					resource.TestCheckResourceAttr(dataSourceName, "mode", "ISCSI"),
				),
			},
		},
	})
}

func TestAccISCSITargetDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_iscsi_target" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
