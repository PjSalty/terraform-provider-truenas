package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccISCSIPortal_basic(t *testing.T) {
	resourceName := "truenas_iscsi_portal.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIPortalDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIPortalConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "comment", "tf-acc-test portal"),
					resource.TestCheckResourceAttrSet(resourceName, "tag"),
					resource.TestCheckResourceAttr(resourceName, "listen.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "listen.0.ip", "0.0.0.0"),
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

func testAccCheckISCSIPortalDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI portal ID not set")
		}
		return nil
	}
}

func testAccISCSIPortalConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_iscsi_portal" "test" {
  comment = "tf-acc-test portal"

  listen = [{
    ip = "0.0.0.0"
  }]
}
`
}
