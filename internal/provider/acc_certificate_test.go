package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// genAccSelfSignedCert produces a PEM-encoded ECDSA P-256 self-signed CA
// certificate and matching private key. Generated fresh for every test run
// so there is no committed key material.
func genAccSelfSignedCert(cn string) (string, string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
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

// TestAccCertificateResource_basic imports a self-signed certificate and
// verifies the provider round-trips name and create_type. privatekey is
// sensitive and excluded from ImportStateVerify.
func TestAccCertificateResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-cert")
	certPEM, keyPEM, err := genAccSelfSignedCert("acctest.example.com")
	if err != nil {
		t.Fatalf("gen self-signed cert: %s", err)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_certificate" "test" {
  name        = %q
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = %q
  privatekey  = %q
}
`, name, certPEM, keyPEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_certificate.test", "name", name),
					resource.TestCheckResourceAttr("truenas_certificate.test", "create_type", "CERTIFICATE_CREATE_IMPORTED"),
					resource.TestCheckResourceAttrSet("truenas_certificate.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_certificate.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"privatekey",
					"create_type",
				},
			},
		},
	})
}

// TestAccCertificateResource_update renames the certificate in place.
// certificate/privatekey/create_type all have RequiresReplace, so they
// must stay identical between steps. The provider attaches a
// PEMEquivalent plan modifier ahead of RequiresReplace on both
// `certificate` and `privatekey` to suppress cosmetic normalization
// diffs that the server re-introduces on read-back (CRLF→LF, base64
// line-wrap width, trailing whitespace). Without that modifier this
// test would fail with "expected Update, got [delete create]".
func TestAccCertificateResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name1 := randomName("acctest-cert-v1")
	name2 := randomName("acctest-cert-v2")
	certPEM, keyPEM, err := genAccSelfSignedCert("acctest-update.example.com")
	if err != nil {
		t.Fatalf("gen self-signed cert: %s", err)
	}
	cfg := func(n string) string {
		return fmt.Sprintf(`
resource "truenas_certificate" "test" {
  name        = %q
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = %q
  privatekey  = %q
}
`, n, certPEM, keyPEM)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(name1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_certificate.test", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_certificate.test", "name", name1),
			},
			{
				Config: cfg(name2),
				// Renaming only the `name` attribute must be an in-place
				// Update (not a destroy+recreate). `certificate`,
				// `privatekey`, and `create_type` carry RequiresReplace,
				// so this assertion guards against any accidental plan
				// modifier drift that would replace the resource.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_certificate.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_certificate.test", "name", name2),
			},
		},
	})
}

// TestAccCertificateResource_disappears deletes the certificate out-of-band
// via the TrueNAS API and verifies the provider detects the drift.
func TestAccCertificateResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctest-cert-gone")
	certPEM, keyPEM, err := genAccSelfSignedCert("acctest-gone.example.com")
	if err != nil {
		t.Fatalf("gen self-signed cert: %s", err)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_certificate" "test" {
  name        = %q
  create_type = "CERTIFICATE_CREATE_IMPORTED"
  certificate = %q
  privatekey  = %q
}
`, name, certPEM, keyPEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_certificate.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_certificate.test"]
						if !ok {
							return fmt.Errorf("truenas_certificate.test not in state")
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
						return c.DeleteCertificate(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
