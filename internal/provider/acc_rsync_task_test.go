package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccRsyncTaskResource_basic creates a MODULE-mode rsync task
// targeting /mnt/test and verifies import + state round-trip.
func TestAccRsyncTaskResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_rsync_task" "test" {
  path         = "/mnt/test"
  user         = "root"
  mode         = "MODULE"
  remotemodule = "acctest"
  remotehost   = "localhost"
  direction    = "PUSH"
  enabled      = false
  desc         = "acctest rsync initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_rsync_task.test", "path", "/mnt/test"),
					resource.TestCheckResourceAttr("truenas_rsync_task.test", "user", "root"),
					resource.TestCheckResourceAttr("truenas_rsync_task.test", "mode", "MODULE"),
					resource.TestCheckResourceAttrSet("truenas_rsync_task.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_rsync_task.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccRsyncTaskResource_disappears verifies that out-of-band deletion
// causes Terraform to detect drift on the next plan.
func TestAccRsyncTaskResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_rsync_task" "test" {
  path         = "/mnt/test"
  user         = "root"
  mode         = "MODULE"
  remotemodule = "acctest"
  remotehost   = "localhost"
  enabled      = false
  desc         = "acctest rsync disappears"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_rsync_task.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_rsync_task.test"]
						if !ok {
							return fmt.Errorf("resource truenas_rsync_task.test not found")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("rsync task id %q not numeric: %w", rs.Primary.ID, err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteRsyncTask(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccRsyncTaskResource_update round-trips the `desc` attribute to
// prove the Update path works end-to-end.
func TestAccRsyncTaskResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	tpl := `
resource "truenas_rsync_task" "test" {
  path         = "/mnt/test"
  user         = "root"
  mode         = "MODULE"
  remotemodule = "acctest"
  remotehost   = "localhost"
  enabled      = false
  desc         = %q
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(tpl, "initial"),
				Check:  resource.TestCheckResourceAttr("truenas_rsync_task.test", "desc", "initial"),
			},
			{
				Config: fmt.Sprintf(tpl, "updated"),
				Check:  resource.TestCheckResourceAttr("truenas_rsync_task.test", "desc", "updated"),
			},
		},
	})
}
