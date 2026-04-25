package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccISCSIInitiator_basic(t *testing.T) {
	resourceName := "truenas_iscsi_initiator.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIInitiatorDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIInitiatorConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "comment", "tf-acc-test initiator"),
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

func testAccCheckISCSIInitiatorDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI initiator ID not set")
		}
		return nil
	}
}

func testAccISCSIInitiatorConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_iscsi_initiator" "test" {
  comment = "tf-acc-test initiator"
}
`
}
