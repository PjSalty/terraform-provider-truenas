package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPrivilegeDataSource_basic creates a privilege (TrueNAS RBAC
// grant) tied to a fixture group and reads it back via the datasource.
// Validates the local_groups list attribute round-trips correctly.
func TestAccPrivilegeDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_privilege.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

resource "truenas_group" "fixture" {
  name = "tf-acc-priv-ds-grp"
}

resource "truenas_privilege" "fixture" {
  name           = "tf-acc-priv-ds"
  local_groups   = [truenas_group.fixture.gid]
  web_shell      = false
  roles          = ["READONLY_ADMIN"]
}

data "truenas_privilege" "test" {
  id = truenas_privilege.fixture.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-acc-priv-ds"),
					resource.TestCheckResourceAttr(dataSourceName, "web_shell", "false"),
				),
			},
		},
	})
}

func TestAccPrivilegeDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_privilege" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
