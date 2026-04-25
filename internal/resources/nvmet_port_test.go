package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNVMetPort_basic(t *testing.T) {
	resourceName := "truenas_nvmet_port.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetPortDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetPortConfigBasic("127.0.0.1", 4420),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "addr_trtype", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "addr_traddr", "127.0.0.1"),
					resource.TestCheckResourceAttr(resourceName, "addr_trsvcid", "4420"),
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

func testAccCheckNVMetPortDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_port ID not set")
		}
		return nil
	}
}

func testAccNVMetPortConfigBasic(addr string, port int) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = %q
  addr_trsvcid = %d
}
`, addr, port)
}
