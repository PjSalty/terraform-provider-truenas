package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFTPConfigResource_basic verifies the singleton FTP config
// resource applies cleanly and imports back with matching state.
func TestAccFTPConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_ftp_config" "test" {
  port    = 21
  clients = 5
  timeout = 600
  banner  = "acctest initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_ftp_config.test", "port", "21"),
					resource.TestCheckResourceAttr("truenas_ftp_config.test", "clients", "5"),
					resource.TestCheckResourceAttrSet("truenas_ftp_config.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_ftp_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccFTPConfigResource_update round-trips the `clients` attribute
// to prove the Update path functions end-to-end.
func TestAccFTPConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_ftp_config" "test" {
  port    = 21
  clients = 5
}
`,
				Check: resource.TestCheckResourceAttr("truenas_ftp_config.test", "clients", "5"),
			},
			{
				Config: `
resource "truenas_ftp_config" "test" {
  port    = 21
  clients = 15
  banner  = "updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_ftp_config.test", "clients", "15"),
					resource.TestCheckResourceAttr("truenas_ftp_config.test", "banner", "updated"),
				),
			},
		},
	})
}
