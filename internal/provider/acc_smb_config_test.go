package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSMBConfigResource_basic verifies the singleton SMB config
// resource applies cleanly and imports back with matching state.
func TestAccSMBConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_smb_config" "test" {
  netbiosname = "truenas"
  workgroup   = "WORKGROUP"
  description = "acc test initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_smb_config.test", "netbiosname", "truenas"),
					resource.TestCheckResourceAttr("truenas_smb_config.test", "workgroup", "WORKGROUP"),
					resource.TestCheckResourceAttrSet("truenas_smb_config.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_smb_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccSMBConfigResource_update round-trips the description attribute
// to prove the Update path functions end-to-end.
func TestAccSMBConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_smb_config" "test" {
  netbiosname = "truenas"
  workgroup   = "WORKGROUP"
  description = "initial"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_smb_config.test", "description", "initial"),
			},
			{
				Config: `
resource "truenas_smb_config" "test" {
  netbiosname = "truenas"
  workgroup   = "WORKGROUP"
  description = "updated"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_smb_config.test", "description", "updated"),
			},
		},
	})
}
