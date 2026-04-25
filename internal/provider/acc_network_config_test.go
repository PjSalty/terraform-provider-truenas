package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNetworkConfigResource_basic — singleton: network config has a
// single instance and Delete is a no-op reset. The test only mutates
// the HTTP proxy field because hostname/domain/gateway changes could
// break the provider's own connection to the TrueNAS API mid-test.
func TestAccNetworkConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_network_config" "test" {
  httpproxy = ""
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_network_config.test", "httpproxy", ""),
					resource.TestCheckResourceAttrSet("truenas_network_config.test", "id"),
				),
			},
		},
	})
}

// TestAccNetworkConfigResource_update toggles the httpproxy field on
// the singleton network config. Third step restores the default of ""
// so the shared test VM is left in a predictable state.
//
// Only httpproxy is mutated because every other field on this resource
// (hostname, domain, gateways, nameservers) risks breaking the
// provider's own connection to the TrueNAS API during the update.
func TestAccNetworkConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_network_config" "test" {
  httpproxy = ""
}
`,
				Check: resource.TestCheckResourceAttr("truenas_network_config.test", "httpproxy", ""),
			},
			{
				Config: `
resource "truenas_network_config" "test" {
  httpproxy = "http://proxy.example.invalid:3128"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_network_config.test", "httpproxy", "http://proxy.example.invalid:3128"),
			},
			{
				// Restore the default (empty) so the shared test VM is
				// left in a predictable state for subsequent runs.
				Config: `
resource "truenas_network_config" "test" {
  httpproxy = ""
}
`,
				Check: resource.TestCheckResourceAttr("truenas_network_config.test", "httpproxy", ""),
			},
		},
	})
}
