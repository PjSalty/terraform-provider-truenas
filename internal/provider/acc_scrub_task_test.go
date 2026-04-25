package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The scrub task resource takes a numeric pool ID, so every test in
// this file stitches in the truenas_pool data source to look up the
// `test` pool's ID at apply time.
const scrubTaskPoolLookup = `
data "truenas_pool" "test" {
  name = "test"
}
`

// TestAccScrubTaskResource_basic creates a scrub task against the test
// pool and verifies import + state round-trip.
func TestAccScrubTaskResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: scrubTaskPoolLookup + `
resource "truenas_scrub_task" "test" {
  pool        = data.truenas_pool.test.id
  threshold   = 35
  description = "acctest scrub initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_scrub_task.test", "threshold", "35"),
					resource.TestCheckResourceAttr("truenas_scrub_task.test", "description", "acctest scrub initial"),
					resource.TestCheckResourceAttrSet("truenas_scrub_task.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_scrub_task.test", "pool_name"),
				),
			},
			{
				ResourceName:      "truenas_scrub_task.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccScrubTaskResource_disappears verifies that out-of-band deletion
// of the scrub task causes Terraform to detect drift on the next plan.
func TestAccScrubTaskResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: scrubTaskPoolLookup + `
resource "truenas_scrub_task" "test" {
  pool        = data.truenas_pool.test.id
  threshold   = 42
  description = "acctest scrub disappears"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_scrub_task.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_scrub_task.test"]
						if !ok {
							return fmt.Errorf("resource truenas_scrub_task.test not found")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("scrub task id %q not numeric: %w", rs.Primary.ID, err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteScrubTask(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccScrubTaskResource_update toggles threshold and description
// across two apply steps to prove the Update path works end-to-end.
func TestAccScrubTaskResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: scrubTaskPoolLookup + `
resource "truenas_scrub_task" "test" {
  pool        = data.truenas_pool.test.id
  threshold   = 35
  description = "initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_scrub_task.test", "threshold", "35"),
					resource.TestCheckResourceAttr("truenas_scrub_task.test", "description", "initial"),
				),
			},
			{
				Config: scrubTaskPoolLookup + `
resource "truenas_scrub_task" "test" {
  pool        = data.truenas_pool.test.id
  threshold   = 60
  description = "updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_scrub_task.test", "threshold", "60"),
					resource.TestCheckResourceAttr("truenas_scrub_task.test", "description", "updated"),
				),
			},
		},
	})
}
