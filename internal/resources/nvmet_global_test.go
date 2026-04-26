package resources_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// testAccCheckNVMetAvailable skips the test if the target TrueNAS doesn't
// expose a reachable NVMe-oF global endpoint. NVMe-oF is optional and may
// not be present on all SCALE versions, so we gate all nvmet acceptance
// tests through this helper.
func testAccCheckNVMetAvailable(t *testing.T) {
	t.Helper()
	url := os.Getenv("TRUENAS_URL")
	key := os.Getenv("TRUENAS_API_KEY")
	if url == "" || key == "" {
		t.Skip("TRUENAS_URL/TRUENAS_API_KEY not set; skipping nvmet test")
	}
	c, err := client.New(url, key)
	if err != nil {
		t.Skipf("cannot create truenas client: %s", err)
	}
	if _, err := c.GetNVMetGlobal(context.Background()); err != nil {
		t.Skipf("nvmet global endpoint not available (nvmet not enabled?): %s", err)
	}
}

func TestAccNVMetGlobal_basic(t *testing.T) {
	resourceName := "truenas_nvmet_global.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetGlobalDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetGlobalConfigBasic("nqn.2014-08.org.nvmexpress:truenas-acc"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "basenqn", "nqn.2014-08.org.nvmexpress:truenas-acc"),
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

func testAccCheckNVMetGlobalDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_global ID not set")
		}
		// Singleton: delete is a state-only no-op.
		return nil
	}
}

func testAccNVMetGlobalConfigBasic(basenqn string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nvmet_global" "test" {
  basenqn = %q
}
`, basenqn)
}
