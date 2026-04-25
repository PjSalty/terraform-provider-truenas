package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNVMetHost_basic(t *testing.T) {
	resourceName := "truenas_nvmet_host.test"
	hostnqn := "nqn.2014-08.org.nvmexpress:uuid:acctest-host-0001"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetHostDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetHostConfigBasic(hostnqn),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "hostnqn", hostnqn),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// dhchap_* secrets are write-only from the API side; skip
				// those attributes in import state verify if needed.
				ImportStateVerifyIgnore: []string{"dhchap_key", "dhchap_ctrl_key"},
			},
		},
	})
}

func testAccCheckNVMetHostDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_host ID not set")
		}
		return nil
	}
}

func testAccNVMetHostConfigBasic(hostnqn string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nvmet_host" "test" {
  hostnqn = %q
}
`, hostnqn)
}
