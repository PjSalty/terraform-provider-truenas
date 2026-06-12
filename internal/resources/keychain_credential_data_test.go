package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccKeychainCredentialDataSource_basic creates an SSH_KEY_PAIR
// keychain credential with a real generated key pair, then reads it
// back via the datasource. The 'attributes' attribute is a
// map(string) — not a JSON-encoded string — so the fixture passes
// literal key strings.
func TestAccKeychainCredentialDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_keychain_credential.test"

	privKey, pubKey, err := generateSSHKeyPair()
	if err != nil {
		t.Fatalf("generate SSH key pair: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

resource "truenas_keychain_credential" "fixture" {
  name = "tf-acc-kc-ds"
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = %q
    public_key  = %q
  }
}

data "truenas_keychain_credential" "test" {
  id = truenas_keychain_credential.fixture.id
}
`, privKey, pubKey),
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
