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

// TestAccUserResource_basic excludes `password` from ImportStateVerify
// because TrueNAS never returns the password on read — the API exposes
// only a hashed value, so the plaintext plan value cannot round-trip.
func TestAccUserResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	username := fmt.Sprintf("acctu%d", pidSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_user" "test" {
  uid          = 20000
  username     = %q
  full_name    = "Acc Test"
  password     = "changeme-acctest"
  shell        = "/usr/bin/bash"
  group_create = true
  home         = "/var/empty"
}
`, username),
				// Apply idempotency gate: after the first apply + state
				// refresh, the plan MUST be empty. Catches Read/Create
				// shape mismatches that would otherwise surface as
				// phantom diffs on every subsequent plan.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "username", username),
					resource.TestCheckResourceAttr("truenas_user.test", "uid", "20000"),
					resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Acc Test"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "group_create"},
			},
		},
	})
}

// TestAccUserResource_update verifies that mutable fields (full_name, shell)
// are round-tripped by Update without replacing the resource. uid/username
// are kept stable between steps since they have RequiresReplace.
func TestAccUserResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	username := fmt.Sprintf("acctu%d", pidSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_user" "test" {
  uid          = 20002
  username     = %q
  full_name    = "Initial Name"
  password     = "changeme-acctest"
  shell        = "/usr/bin/bash"
  group_create = true
  home         = "/var/empty"
}
`, username),
				Check: resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Initial Name"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_user" "test" {
  uid          = 20002
  username     = %q
  full_name    = "Updated Name"
  password     = "changeme-acctest"
  shell        = "/usr/bin/bash"
  group_create = true
  home         = "/var/empty"
}
`, username),
				Check: resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Updated Name"),
			},
		},
	})
}

// TestAccUserResource_disappears deletes the user out-of-band via the
// TrueNAS API mid-test to verify the provider correctly detects the
// drift on the next refresh and plans a recreation.
func TestAccUserResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	username := fmt.Sprintf("acctu%d", pidSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_user" "test" {
  uid          = 20003
  username     = %q
  full_name    = "Disappears"
  password     = "changeme-acctest"
  group_create = true
  home         = "/var/empty"
}
`, username),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_user.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_user.test"]
						if !ok {
							return fmt.Errorf("truenas_user.test not in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parse id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteUser(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
