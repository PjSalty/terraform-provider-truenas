package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccShareNFSDataSource_basic(t *testing.T) {
	pool := testAccDatasetPool()
	dataSourceName := "data.truenas_share_nfs.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "fixture" {
  pool = %q
  name = "tf-acc-nfs-ds-parent"
}

resource "truenas_share_nfs" "fixture" {
  path    = truenas_dataset.fixture.mount_point
  comment = "ds nfs smoke"
}

data "truenas_share_nfs" "test" {
  id = truenas_share_nfs.fixture.id
}
`, pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "comment", "ds nfs smoke"),
				),
			},
		},
	})
}

func TestAccShareNFSDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_share_nfs" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
