package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccService_basic(t *testing.T) {
	resourceName := "truenas_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create (enable SSH service)
			{
				Config: testAccServiceConfig("ssh", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "service", "ssh"),
					resource.TestCheckResourceAttr(resourceName, "enable", "true"),
					resource.TestCheckResourceAttr(resourceName, "state", "RUNNING"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Import by service name
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "ssh",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccService_disableEnable(t *testing.T) {
	resourceName := "truenas_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDestroy(resourceName),
		Steps: []resource.TestStep{
			// Start disabled
			{
				Config: testAccServiceConfig("ssh", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "service", "ssh"),
					resource.TestCheckResourceAttr(resourceName, "enable", "false"),
					resource.TestCheckResourceAttr(resourceName, "state", "STOPPED"),
				),
			},
			// Enable
			{
				Config: testAccServiceConfig("ssh", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable", "true"),
					resource.TestCheckResourceAttr(resourceName, "state", "RUNNING"),
				),
			},
			// Disable again
			{
				Config: testAccServiceConfig("ssh", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable", "false"),
					resource.TestCheckResourceAttr(resourceName, "state", "STOPPED"),
				),
			},
		},
	})
}

func testAccCheckServiceDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("service ID not set")
		}
		return nil
	}
}

func testAccServiceConfig(service string, enable bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_service" "test" {
  service = %q
  enable  = %t
}
`, service, enable)
}
