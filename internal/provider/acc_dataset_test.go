package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccDatasetResource_basic exercises the gold-standard lifecycle:
//
//  1. Create a dataset under the existing `test` pool.
//  2. Read it back and verify core attributes.
//  3. ImportState round-trip — proves the import path and that every
//     attribute round-trips through plain state storage.
//
// This test is skipped unless TF_ACC=1 is set.
func TestAccDatasetResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "test" {
  pool = "test"
  name = %q
}
`, name),
				// Structured plan assertions: verify that the first apply
				// creates the dataset (not updates/replaces) and that
				// known attributes resolve to the expected defaults at
				// plan time. PostApplyPostRefresh enforces the apply
				// idempotency invariant — after a successful apply + a
				// refresh, the next plan MUST be empty. Catches drift
				// from Read returning values the state doesn't hold,
				// default values that round-trip differently, and the
				// whole "terraform plan always says something" family
				// of bugs that plague providers under development.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_dataset.test", plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(
							"truenas_dataset.test",
							tfjsonpath.New("type"),
							knownvalue.StringExact("FILESYSTEM"),
						),
						plancheck.ExpectKnownValue(
							"truenas_dataset.test",
							tfjsonpath.New("pool"),
							knownvalue.StringExact("test"),
						),
						plancheck.ExpectKnownValue(
							"truenas_dataset.test",
							tfjsonpath.New("name"),
							knownvalue.StringExact(name),
						),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.test", "name", name),
					resource.TestCheckResourceAttr("truenas_dataset.test", "pool", "test"),
					resource.TestCheckResourceAttrSet("truenas_dataset.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_dataset.test", "mount_point"),
				),
			},
			{
				ResourceName:      "truenas_dataset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccDatasetResource_disappears verifies that if the dataset is deleted
// out-of-band (e.g., someone runs `zfs destroy` on the box), Terraform
// detects the drift on the next plan and reports a non-empty diff.
func TestAccDatasetResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestds")
	fullID := "test/" + name
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "test" {
  pool = "test"
  name = %q
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.test", "id", fullID),
					func(s *terraform.State) error {
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteDataset(ctx, fullID)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccDatasetResource_update round-trips an updatable attribute
// (comments) to prove the Update path works end-to-end.
func TestAccDatasetResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "test" {
  pool     = "test"
  name     = %q
  comments = "initial"
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_dataset.test", "comments", "initial"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "test" {
  pool        = "test"
  name        = %q
  comments    = "updated"
  compression = "ZSTD"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.test", "comments", "updated"),
					resource.TestCheckResourceAttr("truenas_dataset.test", "compression", "ZSTD"),
				),
			},
		},
	})
}
