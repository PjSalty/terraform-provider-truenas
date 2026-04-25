package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// aclTestConfig stitches together a child dataset + an ACL managed against
// its mount point. The ACL resource requires a filesystem path, so we
// stand up a short-lived dataset to host it and tear both down together.
func aclTestConfig(name string) string {
	return fmt.Sprintf(`
resource "truenas_dataset" "acl_host" {
  pool = "test"
  name = %q
}

resource "truenas_filesystem_acl" "test" {
  path    = truenas_dataset.acl_host.mount_point
  acltype = "POSIX1E"
  uid     = 0
  gid     = 0
  dacl = [
    { tag = "USER_OBJ",  id = -1, perm_read = true, perm_write = true,  perm_execute = true,  default = false },
    { tag = "GROUP_OBJ", id = -1, perm_read = true, perm_write = false, perm_execute = true,  default = false },
    { tag = "OTHER",     id = -1, perm_read = true, perm_write = false, perm_execute = true,  default = false },
  ]
}
`, name)
}

// TestAccFilesystemACLResource_basic creates a child dataset, sets a
// minimal POSIX1E ACL on its mount point, and verifies the attributes
// round-trip through state. Import is not exercised because the ACL
// framework returns normalized entries that can't be cleanly compared.
func TestAccFilesystemACLResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestacl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: aclTestConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_filesystem_acl.test", "acltype", "POSIX1E"),
					resource.TestCheckResourceAttrSet("truenas_filesystem_acl.test", "path"),
					resource.TestCheckResourceAttrSet("truenas_filesystem_acl.test", "id"),
				),
			},
		},
	})
}

// TestAccFilesystemACLResource_disappears simulates out-of-band ACL
// mutation by resetting the ACL on the managed path via the client. The
// next plan must report drift against our recorded DACL.
func TestAccFilesystemACLResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestacl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: aclTestConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_filesystem_acl.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_filesystem_acl.test"]
						if !ok {
							return fmt.Errorf("resource truenas_filesystem_acl.test not found in state")
						}
						path := rs.Primary.Attributes["path"]
						if path == "" {
							return fmt.Errorf("acl path is empty in state")
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						// Overwrite with a DIFFERENT ACL shape so the
						// post-check Read detects meaningful drift against
						// our planned DACL (USER_OBJ rwx → USER_OBJ r-x).
						uid := 0
						gid := 0
						return c.SetFilesystemACL(ctx, &client.SetACLRequest{
							Path:    path,
							ACLType: "POSIX1E",
							UID:     &uid,
							GID:     &gid,
							DACL: []client.SetACLEntry{
								{Tag: "USER_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: false, Execute: true}, Default: false},
								{Tag: "GROUP_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: false, Execute: false}, Default: false},
								{Tag: "OTHER", ID: -1, Perms: client.ACLPerms{Read: false, Write: false, Execute: false}, Default: false},
							},
						})
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccFilesystemACLResource_update round-trips the uid attribute to
// prove the Update path functions end-to-end.
func TestAccFilesystemACLResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestacl")
	baseResource := `
resource "truenas_dataset" "acl_host" {
  pool = "test"
  name = %q
}

resource "truenas_filesystem_acl" "test" {
  path    = truenas_dataset.acl_host.mount_point
  acltype = "POSIX1E"
  uid     = %d
  gid     = 0
  dacl = [
    { tag = "USER_OBJ",  id = -1, perm_read = true, perm_write = true,  perm_execute = true,  default = false },
    { tag = "GROUP_OBJ", id = -1, perm_read = true, perm_write = false, perm_execute = true,  default = false },
    { tag = "OTHER",     id = -1, perm_read = true, perm_write = false, perm_execute = true,  default = false },
  ]
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(baseResource, name, 0),
				Check:  resource.TestCheckResourceAttr("truenas_filesystem_acl.test", "uid", "0"),
			},
			{
				Config: fmt.Sprintf(baseResource, name, 1000),
				Check:  resource.TestCheckResourceAttr("truenas_filesystem_acl.test", "uid", "1000"),
			},
		},
	})
}
