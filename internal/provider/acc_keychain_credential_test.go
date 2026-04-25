package provider

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"golang.org/x/crypto/ssh"
)

// kcGenTestKeyPair generates a fresh ed25519 OpenSSH keypair at runtime.
// Generated rather than committed so secret scanners (GitHub Push Protection,
// gitleaks) do not flag the source tree. TrueNAS does not validate that the
// public and private halves match; it only requires a parseable OpenSSH-format
// private key and authorized_keys-format public key.
func kcGenTestKeyPair(t *testing.T) (privateKey, publicKey string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "acctest")
	if err != nil {
		t.Fatalf("ssh.MarshalPrivateKey: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("ssh.NewPublicKey: %v", err)
	}
	return string(pem.EncodeToMemory(block)), string(ssh.MarshalAuthorizedKey(sshPub))
}

// TestAccKeychainCredentialResource_basic creates a synthetic SSH_KEY_PAIR
// credential. TrueNAS accepts any plausibly-formed OpenSSH key pair; it
// does not verify that the private and public keys actually match. The
// `attributes` map is sensitive and excluded from ImportStateVerify.
func TestAccKeychainCredentialResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-kc")
	priv, pub := kcGenTestKeyPair(t)
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
`, name, priv, pub),
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
	priv, pub := kcGenTestKeyPair(t)
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
`, n, priv, pub)
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
	priv, pub := kcGenTestKeyPair(t)
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
`, name, priv, pub),
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
