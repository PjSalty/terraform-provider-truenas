package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAlertClassesResource_basic — singleton: alert classes config
// has a single instance and Delete is a no-op reset. The classes map
// uses a single well-known alert class (ZpoolCapacityNotice) with its
// default policy so the test does not perturb real alert routing on
// the shared test VM.
func TestAccAlertClassesResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_alertclasses" "test" {
  classes = {
    ZpoolCapacityNotice = {
      level  = "NOTICE"
      policy = "IMMEDIATELY"
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_alertclasses.test", "classes.ZpoolCapacityNotice.level", "NOTICE"),
					resource.TestCheckResourceAttr("truenas_alertclasses.test", "classes.ZpoolCapacityNotice.policy", "IMMEDIATELY"),
					resource.TestCheckResourceAttrSet("truenas_alertclasses.test", "id"),
				),
			},
		},
	})
}

// TestAccAlertClassesResource_update toggles the policy for a single
// known alert class between IMMEDIATELY and HOURLY, then restores it
// to IMMEDIATELY so the shared test VM is left in a predictable state.
// A single alert class is used (ZpoolCapacityNotice) because that
// minimises the risk of colliding with real alert configuration.
func TestAccAlertClassesResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_alertclasses" "test" {
  classes = {
    ZpoolCapacityNotice = {
      level  = "NOTICE"
      policy = "IMMEDIATELY"
    }
  }
}
`,
				Check: resource.TestCheckResourceAttr("truenas_alertclasses.test", "classes.ZpoolCapacityNotice.policy", "IMMEDIATELY"),
			},
			{
				Config: `
resource "truenas_alertclasses" "test" {
  classes = {
    ZpoolCapacityNotice = {
      level  = "NOTICE"
      policy = "HOURLY"
    }
  }
}
`,
				Check: resource.TestCheckResourceAttr("truenas_alertclasses.test", "classes.ZpoolCapacityNotice.policy", "HOURLY"),
			},
			{
				// Restore IMMEDIATELY so the shared test VM is left in
				// a predictable state for subsequent runs.
				Config: `
resource "truenas_alertclasses" "test" {
  classes = {
    ZpoolCapacityNotice = {
      level  = "NOTICE"
      policy = "IMMEDIATELY"
    }
  }
}
`,
				Check: resource.TestCheckResourceAttr("truenas_alertclasses.test", "classes.ZpoolCapacityNotice.policy", "IMMEDIATELY"),
			},
		},
	})
}
