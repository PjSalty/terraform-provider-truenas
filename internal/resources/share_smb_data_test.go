package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShareSMBDataSource_basic(t *testing.T) {
	pool := testAccDatasetPool()
	dataSourceName := "data.truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "fixture" {
  pool = %q
  name = "tf-acc-smb-ds-parent"
}

resource "truenas_share_smb" "fixture" {
  path    = truenas_dataset.fixture.mount_point
  name    = "tf-acc-smb-ds"
  comment = "ds smoke"
}

data "truenas_share_smb" "test" {
  id = truenas_share_smb.fixture.id
}
`, pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-acc-smb-ds"),
					resource.TestCheckResourceAttr(dataSourceName, "comment", "ds smoke"),
				),
			},
		},
	})
}

func TestAccShareSMBDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_share_smb" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
