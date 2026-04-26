package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSNMPConfigResource_basic — singleton: SNMP service
// configuration has a single instance and Delete is a no-op reset.
// Uses community="public" (the TrueNAS default) so the test does not
// leave a custom community string behind.
func TestAccSNMPConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_snmp_config" "test" {
  community = "public"
  contact   = "acctest contact"
  location  = "acctest-rack-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_snmp_config.test", "community", "public"),
					resource.TestCheckResourceAttrSet("truenas_snmp_config.test", "id"),
				),
			},
		},
	})
}

// TestAccSNMPConfigResource_update toggles the contact and location
// fields on the singleton SNMP config. Third step restores the
// defaults ("") so the shared test VM is left in a predictable state.
func TestAccSNMPConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_snmp_config" "test" {
  community = "public"
  contact   = "acctest initial"
  location  = "acctest-rack-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_snmp_config.test", "contact", "acctest initial"),
					resource.TestCheckResourceAttr("truenas_snmp_config.test", "location", "acctest-rack-1"),
				),
			},
			{
				Config: `
resource "truenas_snmp_config" "test" {
  community = "public"
  contact   = "acctest updated"
  location  = "acctest-rack-2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_snmp_config.test", "contact", "acctest updated"),
					resource.TestCheckResourceAttr("truenas_snmp_config.test", "location", "acctest-rack-2"),
				),
			},
			{
				// Restore defaults so the shared test VM is left in a
				// predictable state for subsequent runs.
				Config: `
resource "truenas_snmp_config" "test" {
  community = "public"
  contact   = ""
  location  = ""
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_snmp_config.test", "contact", ""),
					resource.TestCheckResourceAttr("truenas_snmp_config.test", "location", ""),
				),
			},
		},
	})
}
