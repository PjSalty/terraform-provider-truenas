package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSnapshotTaskResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_snapshot_task" "test" {
  dataset        = "test"
  recursive      = false
  lifetime_value = 2
  lifetime_unit  = "WEEK"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_snapshot_task.test", "dataset", "test"),
					resource.TestCheckResourceAttr("truenas_snapshot_task.test", "lifetime_unit", "WEEK"),
					resource.TestCheckResourceAttr("truenas_snapshot_task.test", "lifetime_value", "2"),
					resource.TestCheckResourceAttrSet("truenas_snapshot_task.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_snapshot_task.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccSnapshotTaskResource_disappears verifies that deleting the task
// out-of-band causes Terraform to report drift on the next plan.
func TestAccSnapshotTaskResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_snapshot_task" "test" {
  dataset        = "test"
  recursive      = false
  lifetime_value = 3
  lifetime_unit  = "WEEK"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_snapshot_task.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_snapshot_task.test"]
						if !ok {
							return fmt.Errorf("resource truenas_snapshot_task.test not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("snapshot task ID %q is not numeric: %w", rs.Primary.ID, err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteSnapshotTask(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccSnapshotTaskResource_update changes an updatable attribute
// (lifetime_value) to prove the Update path works end-to-end.
func TestAccSnapshotTaskResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_snapshot_task" "test" {
  dataset        = "test"
  lifetime_value = 2
  lifetime_unit  = "WEEK"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_snapshot_task.test", "lifetime_value", "2"),
			},
			{
				Config: `
resource "truenas_snapshot_task" "test" {
  dataset        = "test"
  lifetime_value = 7
  lifetime_unit  = "DAY"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_snapshot_task.test", "lifetime_value", "7"),
					resource.TestCheckResourceAttr("truenas_snapshot_task.test", "lifetime_unit", "DAY"),
				),
			},
		},
	})
}
