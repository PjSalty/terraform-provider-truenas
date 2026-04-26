package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNVMetSubsys_basic(t *testing.T) {
	resourceName := "truenas_nvmet_subsys.test"
	name := "tf-acc-nvmet-subsys"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetSubsysConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "subnqn"),
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

func testAccCheckNVMetSubsysDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_subsys ID not set")
		}
		return nil
	}
}

func testAccNVMetSubsysConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nvmet_subsys" "test" {
  name = %q
}
`, name)
}
