package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccISCSITarget_basic(t *testing.T) {
	resourceName := "truenas_iscsi_target.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSITargetDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-target"),
					resource.TestCheckResourceAttr(resourceName, "mode", "ISCSI"),
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

func testAccCheckISCSITargetDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI target ID not set")
		}
		return nil
	}
}

func testAccISCSITargetConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_iscsi_portal" "dep" {
  comment = "tf-acc-target-dep portal"

  listen = [{
    ip = "0.0.0.0"
  }]
}

resource "truenas_iscsi_initiator" "dep" {
  comment = "tf-acc-target-dep initiator"
}

resource "truenas_iscsi_target" "test" {
  name = "tf-acc-test-target"

  groups = [{
    portal    = truenas_iscsi_portal.dep.id
    initiator = truenas_iscsi_initiator.dep.id
  }]
}
`
}
