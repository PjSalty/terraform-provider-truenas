package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDirectory_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-dir-test"
	resourceName := "truenas_directory.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryConfigBasic(pool, datasetName, "755"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "mode", "755"),
					resource.TestCheckResourceAttr(resourceName, "uid", "3000"),
					resource.TestCheckResourceAttr(resourceName, "gid", "3000"),
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

func TestAccDirectory_update(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-dir-update"
	resourceName := "truenas_directory.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDirectoryDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryConfigBasic(pool, datasetName, "755"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mode", "755"),
				),
			},
			// mode is applied in place via setperm (no replace).
			{
				Config: testAccDirectoryConfigBasic(pool, datasetName, "750"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mode", "750"),
				),
			},
		},
	})
}

// testAccCheckDirectoryDestroy is intentionally a no-op success check.
// TrueNAS exposes no directory-removal API, so the resource's Delete is
// state-only: the directory is removed from Terraform state but left on
// disk. There is therefore nothing to confirm gone upstream, and a
// CheckDestroy that statted the path would always find it and fail. We
// return nil to record that the destroy step ran without asserting the
// directory was deleted, which would be false.
func testAccCheckDirectoryDestroy(_ string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return nil
	}
}

// uid/gid are set so the live suite exercises the setperm path, not just
// mkdir (the stale-stat bug in issue #21 only shows up after setperm).
func testAccDirectoryConfigBasic(pool, datasetName, mode string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "dir_parent" {
  pool = %q
  name = %q
}

resource "truenas_directory" "test" {
  path = "${truenas_dataset.dir_parent.mount_point}/sub"
  mode = %q
  uid  = 3000
  gid  = 3000
}
`, pool, datasetName, mode)
}
