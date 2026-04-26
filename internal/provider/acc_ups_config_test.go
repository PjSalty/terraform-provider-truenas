package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUPSConfigResource_basic — singleton: UPS daemon configuration
// has a single instance and Delete is a no-op reset. The test uses
// SLAVE mode with a dummy remote host so the TrueNAS test VM never
// tries to drive a physical UPS.
func TestAccUPSConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_ups_config" "test" {
  mode        = "SLAVE"
  identifier  = "ups"
  remotehost  = "127.0.0.1"
  remoteport  = 3493
  description = "acctest"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_ups_config.test", "mode", "SLAVE"),
					resource.TestCheckResourceAttr("truenas_ups_config.test", "description", "acctest"),
					resource.TestCheckResourceAttrSet("truenas_ups_config.test", "id"),
				),
			},
		},
	})
}

// TestAccUPSConfigResource_update toggles the description field on the
// singleton UPS config. Only the description is mutated — every other
// field affects real UPS behaviour and could leave the test VM in a
// broken state. Third step restores the default ("") so the shared
// test VM is left in a predictable state.
func TestAccUPSConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	cfg := func(description string) string {
		return fmt.Sprintf(`
resource "truenas_ups_config" "test" {
  mode        = "SLAVE"
  identifier  = "ups"
  remotehost  = "127.0.0.1"
  remoteport  = 3493
  description = %q
}
`, description)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("acctest-initial"),
				Check:  resource.TestCheckResourceAttr("truenas_ups_config.test", "description", "acctest-initial"),
			},
			{
				Config: cfg("acctest-updated"),
				Check:  resource.TestCheckResourceAttr("truenas_ups_config.test", "description", "acctest-updated"),
			},
			{
				// Restore the default description so the shared test
				// VM is left in a predictable state for subsequent runs.
				Config: cfg(""),
				Check:  resource.TestCheckResourceAttr("truenas_ups_config.test", "description", ""),
			},
		},
	})
}
