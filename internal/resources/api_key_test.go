package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAPIKey_basic(t *testing.T) {
	resourceName := "truenas_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyConfigBasic("tf-acc-test-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-key"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "username"),
				),
			},
		},
	})
}

func TestAccAPIKey_update(t *testing.T) {
	resourceName := "truenas_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyConfigBasic("tf-acc-test-key-v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-key-v1"),
				),
			},
			{
				Config: testAccAPIKeyConfigBasic("tf-acc-test-key-v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-key-v2"),
				),
			},
		},
	})
}

func testAccCheckAPIKeyDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("API key ID not set")
		}
		return nil
	}
}

func testAccAPIKeyConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_api_key" "test" {
  name     = %q
  username = "truenas_admin"
}
`, name)
}
