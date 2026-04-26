package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAlertService_basic(t *testing.T) {
	resourceName := "truenas_alert_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAlertServiceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertServiceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-alert"),
					resource.TestCheckResourceAttr(resourceName, "type", "Mail"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "level", "WARNING"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"settings_json"},
			},
		},
	})
}

func testAccCheckAlertServiceDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("alert service ID not set")
		}
		return nil
	}
}

func testAccAlertServiceConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_alert_service" "test" {
  name  = "tf-acc-test-alert"
  type  = "Mail"
  level = "WARNING"

  settings_json = jsonencode({
    email = "tf-acc-test@example.com"
  })
}
`
}
