package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCatalog_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_CATALOG") == "" {
		t.Skip("Skipping truenas_catalog acceptance test: set TRUENAS_TEST_CATALOG=1. " +
			"This test mutates the singleton catalog preferred_trains on the target TrueNAS.")
	}

	resourceName := "truenas_catalog.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig([]string{"stable"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "label"),
					resource.TestCheckResourceAttrSet(resourceName, "location"),
					resource.TestCheckResourceAttr(resourceName, "preferred_trains.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "preferred_trains.0", "stable"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"sync_on_create"},
			},
		},
	})
}

func testAccCheckCatalogDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("catalog ID not set")
		}
		return nil
	}
}

func testAccCatalogConfig(trains []string) string {
	quoted := ""
	for i, t := range trains {
		if i > 0 {
			quoted += ", "
		}
		quoted += fmt.Sprintf("%q", t)
	}
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_catalog" "test" {
  preferred_trains = [%s]
  sync_on_create   = false
}
`, quoted)
}
