package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUPSConfig_basic(t *testing.T) {
	resourceName := "truenas_ups_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUPSConfigCustom("test-ups", "Test UPS", 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "identifier", "test-ups"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test UPS"),
					resource.TestCheckResourceAttr(resourceName, "shutdowntimer", "60"),
					resource.TestCheckResourceAttr(resourceName, "mode", "MASTER"),
					resource.TestCheckResourceAttr(resourceName, "shutdown", "BATT"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "1",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccUPSConfig_update(t *testing.T) {
	resourceName := "truenas_ups_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUPSConfigCustom("test-ups-2", "Test UPS Device", 45),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", "test-ups-2"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test UPS Device"),
					resource.TestCheckResourceAttr(resourceName, "shutdowntimer", "45"),
				),
			},
			{
				Config: testAccUPSConfigCustom("ups", "", 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "identifier", "ups"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "shutdowntimer", "30"),
				),
			},
		},
	})
}

// testAccUPSConfigCustom uses a valid UPS driver/port combination
// for the test TrueNAS instance.
func testAccUPSConfigCustom(identifier, description string, shutdowntimer int) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_ups_config" "test" {
  identifier    = %q
  description   = %q
  shutdowntimer = %d
  driver        = "usbhid-ups$Back-UPS Pro USB"
  port          = "auto"
}
`, identifier, description, shutdowntimer)
}
