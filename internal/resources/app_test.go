package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccAppCatalogApp returns a catalog app slug known to exist in the
// TRUENAS stable train. Override with TRUENAS_TEST_CATALOG_APP.
func testAccAppCatalogApp() string {
	if v := os.Getenv("TRUENAS_TEST_CATALOG_APP"); v != "" {
		return v
	}
	return "minio"
}

func TestAccApp_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_APPS") == "" {
		t.Skip("Skipping truenas_app acceptance test: set TRUENAS_TEST_APPS=1 and " +
			"ensure the target TrueNAS has the Apps service enabled with a pool configured.")
	}

	catalogApp := testAccAppCatalogApp()
	appName := "tf-acc-app"
	resourceName := "truenas_app.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAppDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAppConfigBasic(appName, catalogApp),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "app_name", appName),
					resource.TestCheckResourceAttr(resourceName, "catalog_app", catalogApp),
					resource.TestCheckResourceAttr(resourceName, "train", "stable"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "state"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"values", "catalog_app", "train", "remove_images", "remove_ix_volumes"},
			},
		},
	})
}

func testAccCheckAppDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("app ID not set")
		}
		return nil
	}
}

func testAccAppConfigBasic(appName, catalogApp string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_app" "test" {
  app_name    = %q
  catalog_app = %q
  train       = "stable"
  version     = "latest"
  values      = "{}"
}
`, appName, catalogApp)
}
