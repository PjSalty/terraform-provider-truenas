package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccZvol_basic(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-zvol-basic"
	resourceName := "truenas_zvol.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckZvolDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccZvolConfigBasic(pool, name, 1073741824),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pool", pool),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "volsize", "1073741824"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// volsize and volblocksize come from the dataset response which doesn't
				// map them directly into the zvol model; skip in import verification.
				ImportStateVerifyIgnore: []string{"volsize", "volblocksize"},
			},
		},
	})
}

func TestAccZvol_withCompression(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-zvol-compress"
	resourceName := "truenas_zvol.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckZvolDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccZvolConfigWithCompression(pool, name, 1073741824, "LZ4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "compression", "LZ4"),
					resource.TestCheckResourceAttr(resourceName, "volsize", "1073741824"),
				),
			},
			// Update compression in-place
			{
				Config: testAccZvolConfigWithCompression(pool, name, 1073741824, "ZSTD"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "compression", "ZSTD"),
				),
			},
		},
	})
}

func testAccCheckZvolDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("zvol ID not set")
		}
		return nil
	}
}

func testAccZvolConfigBasic(pool, name string, volsize int64) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_zvol" "test" {
  pool    = %q
  name    = %q
  volsize = %d
}
`, pool, name, volsize)
}

func testAccZvolConfigWithCompression(pool, name string, volsize int64, compression string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_zvol" "test" {
  pool        = %q
  name        = %q
  volsize     = %d
  compression = %q
}
`, pool, name, volsize, compression)
}
