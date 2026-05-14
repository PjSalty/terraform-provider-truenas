package resources_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// generateSelfSignedCert generates a PEM-encoded self-signed certificate and private key.
func generateSelfSignedCert(cn string) (string, string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return string(certPEM), string(keyPEM), nil
}

func TestAccCertificate_import(t *testing.T) {
	resourceName := "truenas_certificate.test"

	certPEM, keyPEM, err := generateSelfSignedCert("tf-acc-test.example.com")
	if err != nil {
		t.Fatalf("Failed to generate self-signed cert: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCertificateDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateConfigImport("tf-acc-test-cert", certPEM, keyPEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-cert"),
					resource.TestCheckResourceAttr(resourceName, "create_type", "CERTIFICATE_CREATE_IMPORTED"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "dn"),
					resource.TestCheckResourceAttrSet(resourceName, "from"),
					resource.TestCheckResourceAttrSet(resourceName, "until"),
				),
			},
		},
	})
}

func TestAccCertificate_updateName(t *testing.T) {
	resourceName := "truenas_certificate.test"

	certPEM, keyPEM, err := generateSelfSignedCert("tf-acc-test-rename.example.com")
	if err != nil {
		t.Fatalf("Failed to generate self-signed cert: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCertificateDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateConfigImport("tf-acc-test-cert-v1", certPEM, keyPEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-cert-v1"),
				),
			},
			{
				Config: testAccCertificateConfigImport("tf-acc-test-cert-v2", certPEM, keyPEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-cert-v2"),
				),
			},
		},
	})
}

func testAccCheckCertificateDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("certificate ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("certificate ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetCertificate(ctx, id)
		if err == nil {
			return fmt.Errorf("certificate %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of certificate %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckCertificateExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetCertificate(ctx, id); err != nil {
			return fmt.Errorf("certificate %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckCertificateDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteCertificate(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of certificate %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccCertificate_disappears(t *testing.T) {
	cert, key, err := generateSelfSignedCert("disappears.tf-acc.test")
	if err != nil {
		t.Fatalf("generate cert: %v", err)
	}
	name := fmt.Sprintf("tf-acc-cert-disappears-%d", acctest.ShortSuffix())
	resourceName := "truenas_certificate.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCertificateDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateConfigImport(name, cert, key),
				Check:  testAccCheckCertificateExists(resourceName),
			},
			{
				Config:             testAccCertificateConfigImport(name, cert, key),
				Check:              testAccCheckCertificateDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCertificateConfigImport(name, cert, key string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_certificate" "test" {
  name        = %q
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = %q
  privatekey  = %q
}
`, name, cert, key)
}
