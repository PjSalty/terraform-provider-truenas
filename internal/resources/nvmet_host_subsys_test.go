package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNVMetHostSubsys_basic(t *testing.T) {
	resourceName := "truenas_nvmet_host_subsys.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetHostSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetHostSubsysConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "host_id"),
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

func testAccCheckNVMetHostSubsysDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_host_subsys ID not set")
		}
		return nil
	}
}

func testAccNVMetHostSubsysConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_nvmet_host" "hs_host" {
  hostnqn = "nqn.2014-08.org.nvmexpress:uuid:acctest-hs-0001"
}

resource "truenas_nvmet_subsys" "hs_subsys" {
  name = "tf-acc-hs-subsys"
}

resource "truenas_nvmet_host_subsys" "test" {
  host_id   = tonumber(truenas_nvmet_host.hs_host.id)
  subsys_id = tonumber(truenas_nvmet_subsys.hs_subsys.id)
}
`
}
