package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// iscsiTargetExtentPrereqs returns a Terraform snippet that provisions
// a zvol + extent + target, which is the minimum set of resources
// required to exercise the `truenas_iscsi_targetextent` mapping. The
// caller is responsible for creating the mapping itself so each test
// can vary the updatable fields independently.
func iscsiTargetExtentPrereqs(zvol, extent, target string) string {
	return fmt.Sprintf(`
resource "truenas_zvol" "backing" {
  pool    = "test"
  name    = %q
  volsize = 16777216
}

resource "truenas_iscsi_extent" "backing" {
  name = %q
  type = "DISK"
  disk = "zvol/test/${truenas_zvol.backing.name}"
}

resource "truenas_iscsi_target" "backing" {
  name  = %q
  alias = "acctest"
  mode  = "ISCSI"
}
`, zvol, extent, target)
}

// TestAccISCSITargetExtentResource_basic wires together a fresh target
// and extent via a target-extent mapping.
func TestAccISCSITargetExtentResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	zvolName := randomName("accteszvte")
	extentName := fmt.Sprintf("acctestte%d", shortSuffix())
	targetName := fmt.Sprintf("acctestte%d", shortSuffix()+1)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: iscsiTargetExtentPrereqs(zvolName, extentName, targetName) + `
resource "truenas_iscsi_targetextent" "test" {
  target = truenas_iscsi_target.backing.id
  extent = truenas_iscsi_extent.backing.id
  lunid  = 0
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_targetextent.test", "id"),
					resource.TestCheckResourceAttr("truenas_iscsi_targetextent.test", "lunid", "0"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_targetextent.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccISCSITargetExtentResource_update flips the LUN ID in-place.
func TestAccISCSITargetExtentResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	zvolName := randomName("accteszvte")
	extentName := fmt.Sprintf("acctestte%d", shortSuffix())
	targetName := fmt.Sprintf("acctestte%d", shortSuffix()+1)
	prereqs := iscsiTargetExtentPrereqs(zvolName, extentName, targetName)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: prereqs + `
resource "truenas_iscsi_targetextent" "test" {
  target = truenas_iscsi_target.backing.id
  extent = truenas_iscsi_extent.backing.id
  lunid  = 0
}
`,
				Check: resource.TestCheckResourceAttr("truenas_iscsi_targetextent.test", "lunid", "0"),
			},
			{
				Config: prereqs + `
resource "truenas_iscsi_targetextent" "test" {
  target = truenas_iscsi_target.backing.id
  extent = truenas_iscsi_extent.backing.id
  lunid  = 3
}
`,
				Check: resource.TestCheckResourceAttr("truenas_iscsi_targetextent.test", "lunid", "3"),
			},
		},
	})
}

// TestAccISCSITargetExtentResource_disappears deletes the mapping
// out-of-band via the client.
func TestAccISCSITargetExtentResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	zvolName := randomName("accteszvte")
	extentName := fmt.Sprintf("acctestte%d", shortSuffix())
	targetName := fmt.Sprintf("acctestte%d", shortSuffix()+1)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: iscsiTargetExtentPrereqs(zvolName, extentName, targetName) + `
resource "truenas_iscsi_targetextent" "test" {
  target = truenas_iscsi_target.backing.id
  extent = truenas_iscsi_extent.backing.id
  lunid  = 0
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_targetextent.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_iscsi_targetextent.test"]
						if !ok {
							return fmt.Errorf("resource truenas_iscsi_targetextent.test not found in state")
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
						return c.DeleteISCSITargetExtent(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
