package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		return nil
	}
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
