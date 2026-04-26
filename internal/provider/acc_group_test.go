package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccGroupResource_basic creates a local group under a short
// 32-character-limited name. Group names must match the same POSIX
// regex as usernames.
func TestAccGroupResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctg%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_group" "test" {
  gid  = 20000
  name = %q
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_group.test", "name", name),
					resource.TestCheckResourceAttr("truenas_group.test", "gid", "20000"),
					resource.TestCheckResourceAttrSet("truenas_group.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccGroupResource_update toggles the smb flag, which is a pure
// in-place update that does not affect gid/name.
func TestAccGroupResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctg%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_group" "test" {
  gid  = 20010
  name = %q
  smb  = false
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_group.test", "smb", "false"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_group" "test" {
  gid  = 20010
  name = %q
  smb  = true
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_group.test", "smb", "true"),
			},
		},
	})
}

// TestAccGroupResource_disappears deletes the group out-of-band via the
// TrueNAS API to verify the provider detects the drift on refresh.
func TestAccGroupResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctg%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_group" "test" {
  gid  = 20011
  name = %q
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_group.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_group.test"]
						if !ok {
							return fmt.Errorf("truenas_group.test not in state")
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
						return c.DeleteGroup(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
