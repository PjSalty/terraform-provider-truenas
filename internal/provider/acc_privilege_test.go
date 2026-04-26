package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPrivilegeResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-priv")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_privilege" "test" {
  name  = %q
  roles = ["READONLY_ADMIN"]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_privilege.test", "name", name),
					resource.TestCheckResourceAttrSet("truenas_privilege.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_privilege.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPrivilegeResource_update toggles web_shell and the role list
// in place. name stays constant to avoid recreation.
func TestAccPrivilegeResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-priv-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_privilege" "test" {
  name      = %q
  roles     = ["READONLY_ADMIN"]
  web_shell = false
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_privilege.test", "web_shell", "false"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_privilege" "test" {
  name      = %q
  roles     = ["READONLY_ADMIN"]
  web_shell = true
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_privilege.test", "web_shell", "true"),
			},
		},
	})
}

// TestAccPrivilegeResource_disappears deletes the privilege out-of-band.
func TestAccPrivilegeResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-priv-gone")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_privilege" "test" {
  name  = %q
  roles = ["READONLY_ADMIN"]
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_privilege.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_privilege.test"]
						if !ok {
							return fmt.Errorf("truenas_privilege.test not in state")
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
						return c.DeletePrivilege(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
