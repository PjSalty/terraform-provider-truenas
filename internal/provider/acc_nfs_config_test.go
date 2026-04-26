package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNFSConfigResource_basic verifies the singleton NFS config
// resource applies cleanly and imports back with matching state.
func TestAccNFSConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_nfs_config" "test" {
  servers       = 4
  allow_nonroot = false
  protocols     = ["NFSV3", "NFSV4"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nfs_config.test", "servers", "4"),
					resource.TestCheckResourceAttr("truenas_nfs_config.test", "allow_nonroot", "false"),
					resource.TestCheckResourceAttrSet("truenas_nfs_config.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_nfs_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccNFSConfigResource_update changes the server-count attribute
// across two apply steps to prove the Update path functions end-to-end.
func TestAccNFSConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_nfs_config" "test" {
  servers   = 2
  protocols = ["NFSV3", "NFSV4"]
}
`,
				Check: resource.TestCheckResourceAttr("truenas_nfs_config.test", "servers", "2"),
			},
			{
				Config: `
resource "truenas_nfs_config" "test" {
  servers   = 8
  protocols = ["NFSV4"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nfs_config.test", "servers", "8"),
					resource.TestCheckResourceAttr("truenas_nfs_config.test", "protocols.#", "1"),
				),
			},
		},
	})
}
