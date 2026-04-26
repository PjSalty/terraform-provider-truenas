package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// iscsiExtentBackingZvol returns a Terraform snippet that provisions a
// small zvol on the `test` pool so that an iSCSI DISK extent has backing
// storage. The returned zvol name is embedded in the `disk` attribute of
// the extent resource. Each run uses a unique zvol name so concurrent
// runs never collide on the TrueNAS host.
func iscsiExtentBackingZvol(name string) string {
	return fmt.Sprintf(`
resource "truenas_zvol" "backing" {
  pool    = "test"
  name    = %q
  volsize = 16777216
}
`, name)
}

// TestAccISCSIExtentResource_basic creates a DISK-type iSCSI extent
// backed by a freshly-provisioned zvol, then verifies that it can be
// imported.
func TestAccISCSIExtentResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	extentName := fmt.Sprintf("acctestext%d", shortSuffix())
	zvolName := randomName("acctestext-zv")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: iscsiExtentBackingZvol(zvolName) + fmt.Sprintf(`
resource "truenas_iscsi_extent" "test" {
  name = %q
  type = "DISK"
  disk = "zvol/test/${truenas_zvol.backing.name}"
}
`, extentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "name", extentName),
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "type", "DISK"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_extent.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_extent.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccISCSIExtentResource_update toggles the comment / enabled /
// readonly flags in a second apply. All three are updatable without
// requiring replacement.
func TestAccISCSIExtentResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	extentName := fmt.Sprintf("acctestext%d", shortSuffix())
	zvolName := randomName("acctestext-zv")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: iscsiExtentBackingZvol(zvolName) + fmt.Sprintf(`
resource "truenas_iscsi_extent" "test" {
  name     = %q
  type     = "DISK"
  disk     = "zvol/test/${truenas_zvol.backing.name}"
  comment  = "initial"
  enabled  = true
  readonly = false
}
`, extentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "comment", "initial"),
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "enabled", "true"),
				),
			},
			{
				Config: iscsiExtentBackingZvol(zvolName) + fmt.Sprintf(`
resource "truenas_iscsi_extent" "test" {
  name     = %q
  type     = "DISK"
  disk     = "zvol/test/${truenas_zvol.backing.name}"
  comment  = "updated"
  enabled  = false
  readonly = true
}
`, extentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "comment", "updated"),
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "enabled", "false"),
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "readonly", "true"),
				),
			},
		},
	})
}

// TestAccISCSIExtentResource_disappears deletes the extent out-of-band
// via the client and verifies Terraform detects the drift.
func TestAccISCSIExtentResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	extentName := fmt.Sprintf("acctestext%d", shortSuffix())
	zvolName := randomName("acctestext-zv")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: iscsiExtentBackingZvol(zvolName) + fmt.Sprintf(`
resource "truenas_iscsi_extent" "test" {
  name = %q
  type = "DISK"
  disk = "zvol/test/${truenas_zvol.backing.name}"
}
`, extentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_extent.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_iscsi_extent.test"]
						if !ok {
							return fmt.Errorf("resource truenas_iscsi_extent.test not found in state")
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
						return c.DeleteISCSIExtent(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
