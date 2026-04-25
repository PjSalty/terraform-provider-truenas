package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccAPIKeyResource_basic exercises the gold-standard lifecycle.
// The `key` attribute is write-only / returned-on-create and is NOT
// round-tripped through import, so it's excluded from ImportStateVerify.
func TestAccAPIKeyResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestkey")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_api_key" "test" {
  name     = %q
  username = "root"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_api_key.test", "name", name),
					resource.TestCheckResourceAttrSet("truenas_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_api_key.test", "key"),
				),
			},
			{
				ResourceName:            "truenas_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

// TestAccAPIKeyResource_update renames the key in place. `username` has
// RequiresReplace, so it must stay identical between steps.
func TestAccAPIKeyResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name1 := randomName("acctestkey-v1")
	name2 := randomName("acctestkey-v2")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_api_key" "test" {
  name     = %q
  username = "root"
}
`, name1),
				Check: resource.TestCheckResourceAttr("truenas_api_key.test", "name", name1),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_api_key" "test" {
  name     = %q
  username = "root"
}
`, name2),
				Check: resource.TestCheckResourceAttr("truenas_api_key.test", "name", name2),
			},
		},
	})
}

// TestAccAPIKeyResource_disappears deletes the API key out-of-band and
// confirms the provider detects the drift.
func TestAccAPIKeyResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestkey-gone")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_api_key" "test" {
  name     = %q
  username = "root"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_api_key.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_api_key.test"]
						if !ok {
							return fmt.Errorf("truenas_api_key.test not in state")
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
						return c.DeleteAPIKey(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
