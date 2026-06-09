package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// testAccScrubTaskCleanup deletes the existing scrub task for pool 1 if it exists,
// so the acceptance test can create a new one. Uses the live wsclient
// (acctest.Client) for the delete; this is acc-only cleanup, so the
// dial cost is amortised across the full test step.
func testAccScrubTaskCleanup(t *testing.T) {
	t.Helper()
	c, err := acctest.Client()
	if err != nil {
		return // best-effort cleanup
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	// Pool 1's scrub task is the canonical "first acc test fixture"
	// — delete it if it exists. Ignore not-found; it's the expected
	// state on a clean test instance.
	if err := c.DeleteScrubTask(ctx, 1); err != nil && !wsclient.IsNotFound(err) {
		return // best-effort cleanup
	}
}

func TestAccScrubTask_basic(t *testing.T) {
	resourceName := "truenas_scrub_task.test"

	// Delete existing scrub task so we can create a new one
	testAccScrubTaskCleanup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScrubTaskDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccScrubTaskConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "threshold", "35"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "schedule_dow", "7"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckScrubTaskDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("scrub task ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("scrub task ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetScrubTask(ctx, id)
		if err == nil {
			return fmt.Errorf("scrub task %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of scrub task %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckScrubTaskExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetScrubTask(ctx, id); err != nil {
			return fmt.Errorf("scrub task %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckScrubTaskDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteScrubTask(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of scrub task %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccScrubTask_disappears(t *testing.T) {
	testAccScrubTaskCleanup(t)
	resourceName := "truenas_scrub_task.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScrubTaskDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccScrubTaskConfigBasic(),
				Check:  testAccCheckScrubTaskExists(resourceName),
			},
			{
				Config:             testAccScrubTaskConfigBasic(),
				Check:              testAccCheckScrubTaskDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccScrubTaskConfigBasic() string {
	return `
provider "truenas" {}

data "truenas_pool" "test" {
  name = "test"
}

resource "truenas_scrub_task" "test" {
  pool         = data.truenas_pool.test.id
  threshold    = 35
  schedule_dow = "7"
}
`
}
