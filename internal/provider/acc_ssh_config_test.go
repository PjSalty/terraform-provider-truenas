package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSSHConfigResource_basic — singleton: the SSH service
// configuration has a single instance and Delete is a no-op reset.
// Import + destroy are not meaningful for singletons, so the test
// only verifies Create+Read on the default tcpport of 22.
func TestAccSSHConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_ssh_config" "test" {
  tcpport = 22
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_ssh_config.test", "tcpport", "22"),
					resource.TestCheckResourceAttrSet("truenas_ssh_config.test", "id"),
				),
			},
		},
	})
}

// TestAccSSHConfigResource_update toggles tcpport on the singleton SSH
// config. The third step restores the default of 22 so the shared test
// VM is left in a predictable state (and so SSH keeps listening on the
// expected port for any operator who might SSH into the test VM).
//
// No _disappears test exists because this is a singleton: the backend
// has no delete — Delete is a no-op reset — so there is nothing
// meaningful to test for out-of-band deletion drift.
func TestAccSSHConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_ssh_config" "test" {
  tcpport = 22
}
`,
				Check: resource.TestCheckResourceAttr("truenas_ssh_config.test", "tcpport", "22"),
			},
			{
				Config: `
resource "truenas_ssh_config" "test" {
  tcpport = 2222
}
`,
				Check: resource.TestCheckResourceAttr("truenas_ssh_config.test", "tcpport", "2222"),
			},
			{
				// Restore the default port so the shared test VM is
				// left in a predictable state for subsequent runs.
				Config: `
resource "truenas_ssh_config" "test" {
  tcpport = 22
}
`,
				Check: resource.TestCheckResourceAttr("truenas_ssh_config.test", "tcpport", "22"),
			},
		},
	})
}
