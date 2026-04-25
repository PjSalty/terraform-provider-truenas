package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// kcTestPrivateKey is a real ed25519 keypair committed as a test fixture.
// NOT a real credential — generated specifically for this test suite.
// TrueNAS parses the key via libcrypto at create time, so it must parse.
const kcTestPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACDvbMyo62trUjPgD0PFbJnKhDov46BSt6tNE/WTBHqALAAAAJDClmPJwpZj
yQAAAAtzc2gtZWQyNTUxOQAAACDvbMyo62trUjPgD0PFbJnKhDov46BSt6tNE/WTBHqALA
AAAEBxDeIwDmOHlQkv00Y+xoGfnFaFBh4KG5aLylu4QHsf/e9szKjra2tSM+APQ8VsmcqE
Oi/joFK3q00T9ZMEeoAsAAAAB2FjY3Rlc3QBAgMEBQY=
-----END OPENSSH PRIVATE KEY-----
`

const kcTestPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIO9szKjra2tSM+APQ8VsmcqEOi/joFK3q00T9ZMEeoAs acctest"

// TestAccKeychainCredentialResource_basic creates a synthetic SSH_KEY_PAIR
// credential. TrueNAS accepts any plausibly-formed OpenSSH key pair; it
// does not verify that the private and public keys actually match. The
// `attributes` map is sensitive and excluded from ImportStateVerify.
func TestAccKeychainCredentialResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-kc")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_keychain_credential" "test" {
  name = %q
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = %q
    public_key  = %q
  }
}
`, name, kcTestPrivateKey, kcTestPublicKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_keychain_credential.test", "name", name),
					resource.TestCheckResourceAttr("truenas_keychain_credential.test", "type", "SSH_KEY_PAIR"),
					resource.TestCheckResourceAttrSet("truenas_keychain_credential.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_keychain_credential.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"attributes"},
			},
		},
	})
}

// TestAccKeychainCredentialResource_update renames the credential in
// place. `type` has RequiresReplace, so it is held constant.
func TestAccKeychainCredentialResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name1 := randomName("acctest-kc-v1")
	name2 := randomName("acctest-kc-v2")
	cfg := func(n string) string {
		return fmt.Sprintf(`
resource "truenas_keychain_credential" "test" {
  name = %q
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = %q
    public_key  = %q
  }
}
`, n, kcTestPrivateKey, kcTestPublicKey)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(name1),
				Check:  resource.TestCheckResourceAttr("truenas_keychain_credential.test", "name", name1),
			},
			{
				Config: cfg(name2),
				Check:  resource.TestCheckResourceAttr("truenas_keychain_credential.test", "name", name2),
			},
		},
	})
}

// TestAccKeychainCredentialResource_disappears deletes the credential
// out-of-band and verifies drift detection.
func TestAccKeychainCredentialResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-kc-gone")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_keychain_credential" "test" {
  name = %q
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = %q
    public_key  = %q
  }
}
`, name, kcTestPrivateKey, kcTestPublicKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_keychain_credential.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_keychain_credential.test"]
						if !ok {
							return fmt.Errorf("truenas_keychain_credential.test not in state")
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
						return c.DeleteKeychainCredential(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
