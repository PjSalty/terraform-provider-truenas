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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		return nil
	}
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
