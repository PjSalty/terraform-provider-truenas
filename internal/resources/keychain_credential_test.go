package resources_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"golang.org/x/crypto/ssh"
)

// generateSSHKeyPair generates an SSH key pair (OpenSSH-format public key, PEM private key).
func generateSSHKeyPair() (string, string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})

	pubSSH, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubKey := string(ssh.MarshalAuthorizedKey(pubSSH))

	return string(privPEM), pubKey, nil
}

func TestAccKeychainCredential_sshKeyPair(t *testing.T) {
	resourceName := "truenas_keychain_credential.test"

	privKey, pubKey, err := generateSSHKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate SSH key pair: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKeychainCredentialDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccKeychainCredentialConfigSSHKeyPair("tf-acc-test-keypair", privKey, pubKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-keypair"),
					resource.TestCheckResourceAttr(resourceName, "type", "SSH_KEY_PAIR"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccKeychainCredential_updateName(t *testing.T) {
	resourceName := "truenas_keychain_credential.test"

	privKey, pubKey, err := generateSSHKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate SSH key pair: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKeychainCredentialDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccKeychainCredentialConfigSSHKeyPair("tf-acc-test-kc-v1", privKey, pubKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-kc-v1"),
				),
			},
			{
				Config: testAccKeychainCredentialConfigSSHKeyPair("tf-acc-test-kc-v2", privKey, pubKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-kc-v2"),
				),
			},
		},
	})
}

func testAccCheckKeychainCredentialDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("keychain credential ID not set")
		}
		return nil
	}
}

func testAccKeychainCredentialConfigSSHKeyPair(name, privateKey, publicKey string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_keychain_credential" "test" {
  name = %q
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = %q
    public_key  = %q
  }
}
`, name, privateKey, publicKey)
}
