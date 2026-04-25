package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSMBShare_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smb-basic"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccSMBShareConfigBasic(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-smb-basic"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "browsable", "true"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
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

func TestAccSMBShare_update(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smb-update"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create with readonly false
			{
				Config: testAccSMBShareConfigReadOnly(pool, datasetName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
					resource.TestCheckResourceAttr(resourceName, "comment", "acc test share"),
				),
			},
			// Update: set readonly true
			{
				Config: testAccSMBShareConfigReadOnly(pool, datasetName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "readonly", "true"),
				),
			},
		},
	})
}

func testAccCheckSMBShareDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("SMB share ID not set")
		}
		return nil
	}
}

func testAccSMBShareConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "smb_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_smb" "test" {
  path = truenas_dataset.smb_parent.mount_point
  name = %q
}
`, pool, datasetName, datasetName)
}

func testAccSMBShareConfigReadOnly(pool, datasetName string, readOnly bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "smb_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_smb" "test" {
  path     = truenas_dataset.smb_parent.mount_point
  name     = %q
  readonly = %t
  comment  = "acc test share"
}
`, pool, datasetName, datasetName, readOnly)
}
