package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccISCSITargetResource_basic creates a minimal target with no
// groups (groups are optional on TrueNAS SCALE 25.10+).
func TestAccISCSITargetResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctesttgt%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_target" "test" {
  name  = %q
  alias = "test"
  mode  = "ISCSI"
}
`, name),
				// Structured plan assertions. The `groups` block is
				// optional and omitted from the config, so it MUST be
				// null at plan time. This guards against anyone
				// accidentally making `groups` Computed (which would
				// show an unknown value instead).
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_iscsi_target.test", plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(
							"truenas_iscsi_target.test",
							tfjsonpath.New("groups"),
							knownvalue.Null(),
						),
						plancheck.ExpectKnownValue(
							"truenas_iscsi_target.test",
							tfjsonpath.New("mode"),
							knownvalue.StringExact("ISCSI"),
						),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_target.test", "name", name),
					resource.TestCheckResourceAttr("truenas_iscsi_target.test", "alias", "test"),
					resource.TestCheckResourceAttr("truenas_iscsi_target.test", "mode", "ISCSI"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_target.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_target.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccISCSITargetResource_update flips the alias on an existing target
// in-place.
func TestAccISCSITargetResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctesttgt%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_target" "test" {
  name  = %q
  alias = "alias-initial"
  mode  = "ISCSI"
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_iscsi_target.test", "alias", "alias-initial"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_target" "test" {
  name  = %q
  alias = "alias-updated"
  mode  = "ISCSI"
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_iscsi_target.test", "alias", "alias-updated"),
			},
		},
	})
}

// TestAccISCSITargetResource_disappears deletes the target out-of-band and
// expects a non-empty plan afterwards.
func TestAccISCSITargetResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctesttgt%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_target" "test" {
  name  = %q
  alias = "disappears"
  mode  = "ISCSI"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_target.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_iscsi_target.test"]
						if !ok {
							return fmt.Errorf("resource truenas_iscsi_target.test not found in state")
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
						return c.DeleteISCSITarget(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
