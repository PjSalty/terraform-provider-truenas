package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccKeychainCredentialDataSource_basic creates an SSH_KEY_PAIR
// keychain credential (the simplest type to fixture — TrueNAS will
// generate the key pair server-side when we omit the private/public
// material) and reads it back via the datasource.
func TestAccKeychainCredentialDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_keychain_credential.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

resource "truenas_keychain_credential" "fixture" {
  name = "tf-acc-kc-ds"
  type = "SSH_KEY_PAIR"

  attributes = jsonencode({
    private_key = ""
    public_key  = ""
  })
}

data "truenas_keychain_credential" "test" {
  id = truenas_keychain_credential.fixture.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-acc-kc-ds"),
					resource.TestCheckResourceAttr(dataSourceName, "type", "SSH_KEY_PAIR"),
				),
			},
		},
	})
}

func TestAccKeychainCredentialDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_keychain_credential" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
