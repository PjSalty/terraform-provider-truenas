package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNVMetPortSubsys_basic(t *testing.T) {
	resourceName := "truenas_nvmet_port_subsys.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetPortSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetPortSubsysConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "port_id"),
					resource.TestCheckResourceAttrSet(resourceName, "subsys_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckNVMetPortSubsysDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_port_subsys ID not set")
		}
		return nil
	}
}

func testAccNVMetPortSubsysConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_nvmet_port" "ps_port" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = 4420
}

resource "truenas_nvmet_subsys" "ps_subsys" {
  name = "tf-acc-ps-subsys"
}

resource "truenas_nvmet_port_subsys" "test" {
  port_id   = tonumber(truenas_nvmet_port.ps_port.id)
  subsys_id = tonumber(truenas_nvmet_subsys.ps_subsys.id)
}
`
}
