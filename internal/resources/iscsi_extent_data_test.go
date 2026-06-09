package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccISCSIExtentDataSource_basic creates a FILE-type iSCSI extent
// (the simplest variety to materialise — it doesn't need a dedicated
// zvol) and reads it back via the datasource. FILE-type was chosen
// over DISK-type because we don't want to consume a test pool slot.
func TestAccISCSIExtentDataSource_basic(t *testing.T) {
	pool := testAccDatasetPool()
	dataSourceName := "data.truenas_iscsi_extent.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "fixture" {
  pool = %q
  name = "tf-acc-extent-ds-parent"
}

resource "truenas_iscsi_extent" "fixture" {
  name        = "tf-acc-extent-ds"
  type        = "FILE"
  path        = "${truenas_dataset.fixture.mount_point}/extent.img"
  filesize    = 1048576
  blocksize   = 512
  enabled     = true
}

data "truenas_iscsi_extent" "test" {
  id = truenas_iscsi_extent.fixture.id
}
`, pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-acc-extent-ds"),
					resource.TestCheckResourceAttr(dataSourceName, "type", "FILE"),
					resource.TestCheckResourceAttr(dataSourceName, "enabled", "true"),
				),
			},
		},
	})
}

func TestAccISCSIExtentDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_iscsi_extent" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
