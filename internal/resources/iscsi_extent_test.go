package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccISCSIExtent_fileType(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-iscsi-extent-file"
	resourceName := "truenas_iscsi_extent.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIExtentDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIExtentConfigFile(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-extent-file"),
					resource.TestCheckResourceAttr(resourceName, "type", "FILE"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "blocksize", "512"),
					resource.TestCheckResourceAttr(resourceName, "rpm", "SSD"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckISCSIExtentDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI extent ID not set")
		}
		return nil
	}
}

func testAccISCSIExtentConfigFile(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "extent_parent" {
  pool = %q
  name = %q
}

resource "truenas_iscsi_extent" "test" {
  name     = "tf-acc-extent-file"
  type     = "FILE"
  path     = "${truenas_dataset.extent_parent.mount_point}/extent.img"
  filesize = 536870912
}
`, pool, datasetName)
}
