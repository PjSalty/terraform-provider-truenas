package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
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
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetZvol(ctx, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("zvol %q still exists upstream after Terraform removed it", rs.Primary.ID)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of zvol %q: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

func testAccCheckZvolExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetZvol(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("zvol %q should exist but lookup failed: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

func testAccCheckZvolDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteZvol(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("out-of-band delete of zvol %q failed: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

// TestAccZvol_disappears verifies the provider re-creates a zvol that
// was deleted out of band (e.g. via the TrueNAS UI or a direct API call).
// The static counterpart is TestResourcesRemoveFromStateOnNotFound which
// proves the Read handler has the right code shape; this acc test proves
// the wiring works against a live host.
func TestAccZvol_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-zvol-disappears"
	resourceName := "truenas_zvol.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckZvolDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccZvolConfigBasic(pool, name, 1*1024*1024*1024),
				Check:  testAccCheckZvolExists(resourceName),
			},
			{
				Config:             testAccZvolConfigBasic(pool, name, 1*1024*1024*1024),
				Check:              testAccCheckZvolDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
