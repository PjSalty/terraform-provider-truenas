package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFilesystemACL_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-acl-test"
	resourceName := "truenas_filesystem_acl.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFilesystemACLConfigBasic(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "acltype", "POSIX1E"),
					resource.TestCheckResourceAttr(resourceName, "uid", "0"),
					resource.TestCheckResourceAttr(resourceName, "gid", "0"),
					resource.TestCheckResourceAttr(resourceName, "dacl.#", "3"),
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

func testAccFilesystemACLConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "acl_parent" {
  pool = %q
  name = %q
}

resource "truenas_filesystem_acl" "test" {
  path    = truenas_dataset.acl_parent.mount_point
  acltype = "POSIX1E"
  uid     = 0
  gid     = 0

  dacl = [
    {
      tag          = "USER_OBJ"
      id           = -1
      perm_read    = true
      perm_write   = true
      perm_execute = true
      default      = false
    },
    {
      tag          = "GROUP_OBJ"
      id           = -1
      perm_read    = true
      perm_write   = false
      perm_execute = true
      default      = false
    },
    {
      tag          = "OTHER"
      id           = -1
      perm_read    = true
      perm_write   = false
      perm_execute = true
      default      = false
    },
  ]
}
`, pool, datasetName)
}
