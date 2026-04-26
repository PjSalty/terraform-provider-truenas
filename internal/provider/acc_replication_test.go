package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// replicationPrereqs returns a Terraform snippet creating two datasets
// on the `test` pool for LOCAL replication. LOCAL transport does not
// require an SSH credential, which keeps the test hermetic — we simply
// replicate test/<src> into test/<dst>.
func replicationPrereqs(src, dst string) string {
	return fmt.Sprintf(`
resource "truenas_dataset" "src" {
  pool = "test"
  name = %q
}

resource "truenas_dataset" "dst" {
  pool = "test"
  name = %q
}
`, src, dst)
}

// TestAccReplicationResource_basic creates a LOCAL-transport replication
// task between two freshly-minted datasets, then verifies import.
func TestAccReplicationResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestrep%d", shortSuffix())
	src := randomName("acctestrepsrc")
	dst := randomName("acctestrepdst")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: replicationPrereqs(src, dst) + fmt.Sprintf(`
resource "truenas_replication" "test" {
  name                       = %q
  direction                  = "PUSH"
  transport                  = "LOCAL"
  source_datasets            = [truenas_dataset.src.id]
  target_dataset             = truenas_dataset.dst.id
  recursive                  = false
  auto                       = false
  enabled                    = true
  retention_policy           = "NONE"
  also_include_naming_schema = ["auto-%%Y-%%m-%%d_%%H-%%M"]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_replication.test", "name", name),
					resource.TestCheckResourceAttr("truenas_replication.test", "direction", "PUSH"),
					resource.TestCheckResourceAttr("truenas_replication.test", "transport", "LOCAL"),
					resource.TestCheckResourceAttrSet("truenas_replication.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_replication.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccReplicationResource_update toggles the `enabled` flag in place.
func TestAccReplicationResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestrep%d", shortSuffix())
	src := randomName("acctestrepsrc")
	dst := randomName("acctestrepdst")
	prereqs := replicationPrereqs(src, dst)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: prereqs + fmt.Sprintf(`
resource "truenas_replication" "test" {
  name                       = %q
  direction                  = "PUSH"
  transport                  = "LOCAL"
  source_datasets            = [truenas_dataset.src.id]
  target_dataset             = truenas_dataset.dst.id
  auto                       = false
  enabled                    = true
  retention_policy           = "NONE"
  also_include_naming_schema = ["auto-%%Y-%%m-%%d_%%H-%%M"]
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_replication.test", "enabled", "true"),
			},
			{
				Config: prereqs + fmt.Sprintf(`
resource "truenas_replication" "test" {
  name                       = %q
  direction                  = "PUSH"
  transport                  = "LOCAL"
  source_datasets            = [truenas_dataset.src.id]
  target_dataset             = truenas_dataset.dst.id
  auto                       = false
  enabled                    = false
  retention_policy           = "NONE"
  also_include_naming_schema = ["auto-%%Y-%%m-%%d_%%H-%%M"]
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_replication.test", "enabled", "false"),
			},
		},
	})
}

// TestAccReplicationResource_disappears deletes the replication task
// out-of-band via the client.
func TestAccReplicationResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestrep%d", shortSuffix())
	src := randomName("acctestrepsrc")
	dst := randomName("acctestrepdst")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: replicationPrereqs(src, dst) + fmt.Sprintf(`
resource "truenas_replication" "test" {
  name                       = %q
  direction                  = "PUSH"
  transport                  = "LOCAL"
  source_datasets            = [truenas_dataset.src.id]
  target_dataset             = truenas_dataset.dst.id
  auto                       = false
  enabled                    = true
  retention_policy           = "NONE"
  also_include_naming_schema = ["auto-%%Y-%%m-%%d_%%H-%%M"]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_replication.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_replication.test"]
						if !ok {
							return fmt.Errorf("resource truenas_replication.test not found in state")
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
						return c.DeleteReplication(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
