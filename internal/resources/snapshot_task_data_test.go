package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSnapshotTaskDataSource_basic creates a snapshot task via the
// resource path then reads it via the datasource. Validates the
// schedule_* attributes (cron-style) and the lifetime fields round-
// trip with the same JSON shape.
func TestAccSnapshotTaskDataSource_basic(t *testing.T) {
	pool := testAccDatasetPool()
	dataSourceName := "data.truenas_snapshot_task.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "fixture" {
  pool = %q
  name = "tf-acc-snap-ds-parent"
}

resource "truenas_snapshot_task" "fixture" {
  dataset         = truenas_dataset.fixture.id
  recursive       = true
  lifetime_value  = 2
  lifetime_unit   = "WEEK"
  naming_schema   = "auto-%%Y%%m%%d.%%H%%M%%S"
  enabled         = true
  schedule_minute = "0"
  schedule_hour   = "3"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"
}

data "truenas_snapshot_task" "test" {
  id = truenas_snapshot_task.fixture.id
}
`, pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "recursive", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "lifetime_value", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "lifetime_unit", "WEEK"),
					resource.TestCheckResourceAttr(dataSourceName, "schedule_hour", "3"),
				),
			},
		},
	})
}

func TestAccSnapshotTaskDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_snapshot_task" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
