package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccSnapshotTask_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-snap-task"
	resourceName := "truenas_snapshot_task.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSnapshotTaskDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotTaskConfigBasic(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "recursive", "false"),
					resource.TestCheckResourceAttr(resourceName, "lifetime_value", "2"),
					resource.TestCheckResourceAttr(resourceName, "lifetime_unit", "WEEK"),
					resource.TestCheckResourceAttr(resourceName, "schedule_minute", "0"),
					resource.TestCheckResourceAttr(resourceName, "schedule_hour", "0"),
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

func testAccCheckSnapshotTaskDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("snapshot task ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("snapshot task ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetSnapshotTask(ctx, id)
		if err == nil {
			return fmt.Errorf("snapshot task %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of snapshot task %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckSnapshotTaskExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetSnapshotTask(ctx, id); err != nil {
			return fmt.Errorf("snapshot task %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckSnapshotTaskDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteSnapshotTask(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of snapshot task %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccSnapshotTask_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-snaptask-disappears"
	resourceName := "truenas_snapshot_task.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSnapshotTaskDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotTaskConfigBasic(pool, datasetName),
				Check:  testAccCheckSnapshotTaskExists(resourceName),
			},
			{
				Config:             testAccSnapshotTaskConfigBasic(pool, datasetName),
				Check:              testAccCheckSnapshotTaskDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccSnapshotTaskConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "snap_parent" {
  pool = %q
  name = %q
}

resource "truenas_snapshot_task" "test" {
  dataset        = "${truenas_dataset.snap_parent.pool}/${truenas_dataset.snap_parent.name}"
  lifetime_value = 2
  lifetime_unit  = "WEEK"
  schedule_minute = "0"
  schedule_hour   = "0"
}
`, pool, datasetName)
}
