package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFilesystemACLTemplate_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_ACLTEMPLATE") != "1" {
		t.Skip("TRUENAS_TEST_ACLTEMPLATE=1 not set; skipping")
	}

	resourceName := "truenas_filesystem_acl_template.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
