package resources_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccFilesystemACLTemplate_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_ACLTEMPLATE") != "1" {
		t.Skip("TRUENAS_TEST_ACLTEMPLATE=1 not set; skipping")
	}

	resourceName := "truenas_filesystem_acl_template.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckFilesystemACLTemplateDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccFilesystemACLTemplateConfigBasic("tf-acc-acltmpl"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-acltmpl"),
					resource.TestCheckResourceAttr(resourceName, "acltype", "POSIX1E"),
					resource.TestCheckResourceAttr(resourceName, "builtin", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccFilesystemACLTemplateConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_filesystem_acl_template" "test" {
  name    = %q
  acltype = "POSIX1E"
  comment = "tf-acc-test"
  acl_json = jsonencode([
    { tag = "USER_OBJ",  id = -1, perms = { READ = true, WRITE = true, EXECUTE = true }, default = false },
    { tag = "GROUP_OBJ", id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false },
    { tag = "OTHER",     id = -1, perms = { READ = true, WRITE = false, EXECUTE = true }, default = false }
  ])
}
`, name)
}

// testAccCheckFilesystemACLTemplateDestroy verifies the ACL template
// is gone from the upstream after Terraform removes it.
func testAccCheckFilesystemACLTemplateDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("filesystem_acl_template ID not set on %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("filesystem_acl_template ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetFilesystemACLTemplate(ctx, id)
		if err == nil {
			return fmt.Errorf("filesystem_acl_template %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of filesystem_acl_template %d: %w", id, err)
		}
		return nil
	}
}
