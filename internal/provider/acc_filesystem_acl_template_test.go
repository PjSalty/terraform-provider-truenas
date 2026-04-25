package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccFilesystemACLTemplateResource_basic creates a minimal POSIX1E
// ACL template. The acl_json payload is excluded from ImportStateVerify
// because TrueNAS normalizes the serialized form on round-trip.
func TestAccFilesystemACLTemplateResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-acl-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_filesystem_acl_template" "test" {
  name    = %q
  acltype = "POSIX1E"
  comment = "acctest"
  acl_json = jsonencode([
    { tag = "USER_OBJ", id = -1, perms = { READ = true, WRITE = true, EXECUTE = true }, default = false },
    { tag = "GROUP_OBJ", id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false },
    { tag = "OTHER", id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false },
  ])
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_filesystem_acl_template.test", "name", name),
					resource.TestCheckResourceAttr("truenas_filesystem_acl_template.test", "acltype", "POSIX1E"),
					resource.TestCheckResourceAttrSet("truenas_filesystem_acl_template.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_filesystem_acl_template.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"acl_json"},
			},
		},
	})
}

// TestAccFilesystemACLTemplateResource_disappears verifies that deleting
// the template out-of-band causes Terraform to detect drift on next plan.
func TestAccFilesystemACLTemplateResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-acl-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_filesystem_acl_template" "test" {
  name    = %q
  acltype = "POSIX1E"
  acl_json = jsonencode([
    { tag = "USER_OBJ", id = -1, perms = { READ = true, WRITE = true, EXECUTE = true }, default = false },
    { tag = "GROUP_OBJ", id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false },
    { tag = "OTHER", id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false },
  ])
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_filesystem_acl_template.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_filesystem_acl_template.test"]
						if !ok {
							return fmt.Errorf("resource truenas_filesystem_acl_template.test not found")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("template id %q not numeric: %w", rs.Primary.ID, err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteFilesystemACLTemplate(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccFilesystemACLTemplateResource_update round-trips the `comment`
// attribute to prove the Update path functions end-to-end.
func TestAccFilesystemACLTemplateResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-acl-%d", shortSuffix())
	tpl := `
resource "truenas_filesystem_acl_template" "test" {
  name    = %q
  acltype = "POSIX1E"
  comment = %q
  acl_json = jsonencode([
    { tag = "USER_OBJ", id = -1, perms = { READ = true, WRITE = true, EXECUTE = true }, default = false },
    { tag = "GROUP_OBJ", id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false },
    { tag = "OTHER", id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false },
  ])
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(tpl, name, "initial"),
				Check:  resource.TestCheckResourceAttr("truenas_filesystem_acl_template.test", "comment", "initial"),
			},
			{
				Config: fmt.Sprintf(tpl, name, "updated"),
				Check:  resource.TestCheckResourceAttr("truenas_filesystem_acl_template.test", "comment", "updated"),
			},
		},
	})
}
