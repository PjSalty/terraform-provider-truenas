package resources_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"golang.org/x/crypto/ssh"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
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
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("keychain credential ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetKeychainCredential(ctx, id)
		if err == nil {
			return fmt.Errorf("keychain credential %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of keychain credential %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckKeychainCredentialExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetKeychainCredential(ctx, id); err != nil {
			return fmt.Errorf("keychain credential %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckKeychainCredentialDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteKeychainCredential(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of keychain credential %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccKeychainCredential_disappears(t *testing.T) {
	resourceName := "truenas_keychain_credential.test"
	priv, pub, err := generateSSHKeyPair()
	if err != nil {
		t.Fatalf("generate ssh key pair: %v", err)
	}
	name := fmt.Sprintf("tf-acc-keychain-disappears-%d", acctest.ShortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKeychainCredentialDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccKeychainCredentialConfigSSHKeyPair(name, priv, pub),
				Check:  testAccCheckKeychainCredentialExists(resourceName),
			},
			{
				Config:             testAccKeychainCredentialConfigSSHKeyPair(name, priv, pub),
				Check:              testAccCheckKeychainCredentialDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
