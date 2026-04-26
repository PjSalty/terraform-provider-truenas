package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNVMetGlobalResource_basic — singleton resource. Only one
// instance exists on a TrueNAS SCALE system and Delete is a state-only
// removal, so only Create+Read and Update paths are meaningful. Import
// round-trips the ID ("1").
func TestAccNVMetGlobalResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_nvmet_global" "test" {
  ana = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_global.test", "id"),
					resource.TestCheckResourceAttr("truenas_nvmet_global.test", "ana", "false"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_global.test", "basenqn"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_global.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccNVMetGlobalResource_update toggles `xport_referral` to verify
// the PUT /nvmet/global round-trip. The `ana` attribute is intentionally
// NOT toggled — Asymmetric Namespace Access requires multi-controller
// hardware support and SCALE rejects ana=true with HTTP 422 on
// single-node test VMs ("This platform does not support Asymmetric
// Namespace Access(ANA).").
func TestAccNVMetGlobalResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_nvmet_global" "test" {
  ana            = false
  xport_referral = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_global.test", "ana", "false"),
					resource.TestCheckResourceAttr("truenas_nvmet_global.test", "xport_referral", "false"),
				),
			},
			{
				Config: `
resource "truenas_nvmet_global" "test" {
  ana            = false
  xport_referral = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_global.test", "ana", "false"),
					resource.TestCheckResourceAttr("truenas_nvmet_global.test", "xport_referral", "true"),
				),
			},
		},
	})
}
