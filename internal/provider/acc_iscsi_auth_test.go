package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccISCSIAuthResource_basic — the secret is sensitive and excluded
// from ImportStateVerify because TrueNAS does not return it on read.
func TestAccISCSIAuthResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_auth" "test" {
  tag    = 100
  user   = "acctest"
  secret = "secretpass1234"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "tag", "100"),
					resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "user", "acctest"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_auth.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_iscsi_auth.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "peersecret"},
			},
		},
	})
}

// TestAccISCSIAuthResource_update changes the user/secret in place while
// keeping the tag stable.
func TestAccISCSIAuthResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	// A unique tag per run avoids collisions with parallel CI runs or
	// leftover state on the shared TrueNAS test VM. Tag is a small int,
	// so we fold a shortSuffix() into the 1..32767 range.
	tag := (shortSuffix() % 30000) + 100
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_auth" "test" {
  tag    = %d
  user   = "acctest1"
  secret = "secretpass1234"
}
`, tag),
				Check: resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "user", "acctest1"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_auth" "test" {
  tag    = %d
  user   = "acctest2"
  secret = "othersecret123"
}
`, tag),
				Check: resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "user", "acctest2"),
			},
		},
	})
}

// TestAccISCSIAuthResource_disappears deletes the auth credential
// out-of-band.
func TestAccISCSIAuthResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	tag := (shortSuffix() % 30000) + 100
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_auth" "test" {
  tag    = %d
  user   = "acctestd"
  secret = "secretpass1234"
}
`, tag),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_auth.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_iscsi_auth.test"]
						if !ok {
							return fmt.Errorf("resource truenas_iscsi_auth.test not found in state")
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
						return c.DeleteISCSIAuth(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
