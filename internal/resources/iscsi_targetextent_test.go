package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccISCSITargetExtent_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-iscsi-te"
	resourceName := "truenas_iscsi_targetextent.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSITargetExtentDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetExtentConfigBasic(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "target"),
					resource.TestCheckResourceAttrSet(resourceName, "extent"),
					resource.TestCheckResourceAttrSet(resourceName, "lunid"),
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

func testAccCheckISCSITargetExtentDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI target-extent ID not set")
		}
		return nil
	}
}

func testAccISCSITargetExtentConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "te_parent" {
  pool = %q
  name = %q
}

resource "truenas_iscsi_extent" "te_extent" {
  name     = "tf-acc-te-extent"
  type     = "FILE"
  path     = "${truenas_dataset.te_parent.mount_point}/te-extent.img"
  filesize = 536870912
}

resource "truenas_iscsi_target" "te_target" {
  name = "tf-acc-te-target"
  mode = "ISCSI"
}

resource "truenas_iscsi_targetextent" "test" {
  target = truenas_iscsi_target.te_target.id
  extent = truenas_iscsi_extent.te_extent.id
}
`, pool, datasetName)
}
