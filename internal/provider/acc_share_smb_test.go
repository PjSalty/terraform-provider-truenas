package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccShareSMBResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestsmb")
	share := randomName("smbshare")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "smb_backing" {
  pool       = "test"
  name       = %q
  share_type = "SMB"
}

resource "truenas_share_smb" "test" {
  path    = truenas_dataset.smb_backing.mount_point
  name    = %q
  enabled = true
}
`, ds, share),
				// Apply idempotency gate: after the first apply + state
				// refresh, the plan MUST be empty. SMB shares have many
				// computed config fields that TrueNAS can set to server
				// defaults; this guards against Read returning those
				// defaults as diffable "changes" on every plan.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_smb.test", "name", share),
					resource.TestCheckResourceAttr("truenas_share_smb.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("truenas_share_smb.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_share_smb.test", "path"),
				),
			},
			{
				ResourceName:      "truenas_share_smb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccShareSMBResource_update toggles browsable/readonly/comment
// in-place to exercise the Update path.
func TestAccShareSMBResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestsmb")
	share := randomName("smbshare")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "smb_backing" {
  pool       = "test"
  name       = %q
  share_type = "SMB"
}

resource "truenas_share_smb" "test" {
  path      = truenas_dataset.smb_backing.mount_point
  name      = %q
  comment   = "initial"
  enabled   = true
  browsable = true
  readonly  = false
}
`, ds, share),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_smb.test", "comment", "initial"),
					resource.TestCheckResourceAttr("truenas_share_smb.test", "readonly", "false"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "smb_backing" {
  pool       = "test"
  name       = %q
  share_type = "SMB"
}

resource "truenas_share_smb" "test" {
  path      = truenas_dataset.smb_backing.mount_point
  name      = %q
  comment   = "updated"
  enabled   = false
  browsable = false
  readonly  = true
}
`, ds, share),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_smb.test", "comment", "updated"),
					resource.TestCheckResourceAttr("truenas_share_smb.test", "enabled", "false"),
					resource.TestCheckResourceAttr("truenas_share_smb.test", "browsable", "false"),
					resource.TestCheckResourceAttr("truenas_share_smb.test", "readonly", "true"),
				),
			},
		},
	})
}

// TestAccShareSMBResource_disappears verifies out-of-band drift recovery.
func TestAccShareSMBResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestsmb")
	share := randomName("smbshare")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "smb_backing" {
  pool       = "test"
  name       = %q
  share_type = "SMB"
}

resource "truenas_share_smb" "test" {
  path    = truenas_dataset.smb_backing.mount_point
  name    = %q
  enabled = true
}
`, ds, share),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_share_smb.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_share_smb.test"]
						if !ok {
							return fmt.Errorf("resource truenas_share_smb.test not found in state")
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
						return c.DeleteSMBShare(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
