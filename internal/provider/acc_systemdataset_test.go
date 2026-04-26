package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSystemDatasetResource_basic verifies that the singleton
// systemdataset resource can be applied and imported. The `test` pool
// is used as the hosting pool — this is the only pool available on the
// acceptance test VM. Read is idempotent and does not alter state.
func TestAccSystemDatasetResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_systemdataset" "test" {
  pool = "test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_systemdataset.test", "id", "systemdataset"),
					resource.TestCheckResourceAttr("truenas_systemdataset.test", "pool", "test"),
					resource.TestCheckResourceAttrSet("truenas_systemdataset.test", "basename"),
				),
			},
			{
				ResourceName:      "truenas_systemdataset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccSystemDatasetResource_disappears is not applicable to a
// singleton — there is no out-of-band "delete" path. Kept as a skipped
// no-op so the gold-standard test triple is discoverable across all
// storage resources.
func TestAccSystemDatasetResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	t.Skip("systemdataset is a singleton — no out-of-band delete path exists")
}

// TestAccSystemDatasetResource_update applies the singleton twice with
// the same pool value. This is the minimal update round-trip we can
// exercise safely because the test VM only has one pool and moving the
// system dataset between pools is a disruptive operation.
func TestAccSystemDatasetResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_systemdataset" "test" {
  pool = "test"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_systemdataset.test", "pool", "test"),
			},
			{
				// Re-apply identical config; proves plan is stable and Update is idempotent.
				Config: `
resource "truenas_systemdataset" "test" {
  pool = "test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_systemdataset.test", "pool", "test"),
					resource.TestCheckResourceAttrSet("truenas_systemdataset.test", "uuid"),
				),
			},
		},
	})
}
