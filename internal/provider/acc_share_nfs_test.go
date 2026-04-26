package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccShareNFSResource_basic creates a short-lived dataset so that the
// NFS share has a valid path to export. The depends_on is implicit via
// interpolation.
func TestAccShareNFSResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestnfs")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "nfs_backing" {
  pool = "test"
  name = %q
}

resource "truenas_share_nfs" "test" {
  path    = truenas_dataset.nfs_backing.mount_point
  comment = "acctest"
  enabled = true
}
`, ds),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_nfs.test", "comment", "acctest"),
					resource.TestCheckResourceAttr("truenas_share_nfs.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("truenas_share_nfs.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_share_nfs.test", "path"),
				),
			},
			{
				ResourceName:      "truenas_share_nfs.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccShareNFSResource_update flips the comment and enabled flag in a
// second apply step and verifies Terraform picks up the update without a
// forced replacement.
func TestAccShareNFSResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestnfs")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "nfs_backing" {
  pool = "test"
  name = %q
}

resource "truenas_share_nfs" "test" {
  path    = truenas_dataset.nfs_backing.mount_point
  comment = "initial"
  enabled = true
}
`, ds),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_nfs.test", "comment", "initial"),
					resource.TestCheckResourceAttr("truenas_share_nfs.test", "enabled", "true"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "nfs_backing" {
  pool = "test"
  name = %q
}

resource "truenas_share_nfs" "test" {
  path    = truenas_dataset.nfs_backing.mount_point
  comment = "updated"
  enabled = false
}
`, ds),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_nfs.test", "comment", "updated"),
					resource.TestCheckResourceAttr("truenas_share_nfs.test", "enabled", "false"),
				),
			},
		},
	})
}

// TestAccShareNFSResource_disappears deletes the NFS share out-of-band via
// the client and verifies Terraform detects the drift — a non-empty plan
// must follow.
func TestAccShareNFSResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestnfs")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "nfs_backing" {
  pool = "test"
  name = %q
}

resource "truenas_share_nfs" "test" {
  path    = truenas_dataset.nfs_backing.mount_point
  enabled = true
}
`, ds),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_share_nfs.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_share_nfs.test"]
						if !ok {
							return fmt.Errorf("resource truenas_share_nfs.test not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("bad id %q: %w", rs.Primary.ID, err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteNFSShare(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
