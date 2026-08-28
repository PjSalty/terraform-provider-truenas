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
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-cert-v1"),
				),
			},
			{
				Config: testAccCertificateConfigImport("tf-acc-test-cert-v2", certPEM, keyPEM),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
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
		if !wsclient.IsNotFound(err) {
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: testAccCheckCertificateExists(resourceName),
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

func testAccCertificateConfigCSR(name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_certificate" "test" {
  name             = %q
  create_type      = "CERTIFICATE_CREATE_CSR"
  common           = "csr.tf-acc.test"
  san              = ["csr.tf-acc.test", "alt.tf-acc.test"]
  key_type         = "RSA"
  key_length       = 2048
  digest_algorithm = "SHA256"
  country          = "US"
  state            = "CA"
  city             = "San Jose"
  organization     = "tf-acc"
  email            = "tf-acc@example.com"
}
`, name)
}

// A CSR is what GitHub issue 33 was actually blocked on twice over: the
// create_type was unusable without a san, and the san that did work came back
// from the server as DNS:csr.tf-acc.test and reported a permanent diff. Both
// have to hold in one run, so the empty-plan check is the assertion.
func TestAccCertificate_csr(t *testing.T) {
	name := acctest.RandomName("tfacccsr")
	resourceName := "truenas_certificate.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCertificateDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateConfigCSR(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCertificateExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "create_type", "CERTIFICATE_CREATE_CSR"),
					resource.TestCheckResourceAttr(resourceName, "key_type", "RSA"),
					resource.TestCheckResourceAttr(resourceName, "key_length", "2048"),
					// The bare name the config asked for, matched against
					// whatever kind the server attached to it.
					resource.TestCheckResourceAttr(resourceName, "san.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "san.0", "csr.tf-acc.test"),
					resource.TestCheckResourceAttr(resourceName, "san.1", "alt.tf-acc.test"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// ImportState hardcodes create_type=CERTIFICATE_CREATE_IMPORTED
				// because the API does not report how a certificate was made,
				// and privatekey is never returned by the API at all.
				//
				// digest_algorithm and san are the server's own answers, not a
				// Read-side bug: certificate.query reports digest_algorithm as
				// "" for a CSR (there is no signed certificate to read it off)
				// and renders each san with its general-name kind attached.
				// ImportStateVerify compares raw state strings, so it does not
				// run the semantic equality that makes DNS:example.com and
				// example.com the same value everywhere else.
				ImportStateVerifyIgnore: []string{
					"create_type", "privatekey", "timeouts",
					"digest_algorithm", "san",
				},
			},
		},
	})
}

// These are plan-time rules, so they cost one plan each rather than a create.
// Without them the failure arrives from middleware as an EINVAL naming a field
// the practitioner could not set, which is what the issue reported.
func TestAccCertificate_planTimeValidation(t *testing.T) {
	cases := []struct {
		name   string
		config string
		expect string
	}{
		{
			name: "csr without san",
			config: `
provider "truenas" {}
resource "truenas_certificate" "test" {
  name        = "tf-acc-invalid"
  create_type = "CERTIFICATE_CREATE_CSR"
  common      = "example.com"
}`,
			expect: `Missing SAN`,
		},
		{
			name: "rsa csr without key_length",
			config: `
provider "truenas" {}
resource "truenas_certificate" "test" {
  name        = "tf-acc-invalid"
  create_type = "CERTIFICATE_CREATE_CSR"
  san         = ["example.com"]
  key_type    = "RSA"
}`,
			expect: `Missing key_length`,
		},
		{
			name: "acme without its fields",
			config: `
provider "truenas" {}
resource "truenas_certificate" "test" {
  name        = "tf-acc-invalid"
  create_type = "CERTIFICATE_CREATE_ACME"
}`,
			expect: `Missing tos`,
		},
		{
			name: "acme field on a non-acme certificate",
			config: `
provider "truenas" {}
resource "truenas_certificate" "test" {
  name        = "tf-acc-invalid"
  create_type = "CERTIFICATE_CREATE_CSR"
  san         = ["example.com"]
  key_type    = "EC"
  renew_days  = 10
}`,
			expect: `Invalid renew_days`,
		},
		{
			name: "renew_days outside the range middleware accepts",
			config: `
provider "truenas" {}
resource "truenas_certificate" "test" {
  name               = "tf-acc-invalid"
  create_type        = "CERTIFICATE_CREATE_ACME"
  tos                = true
  csr_id             = 1
  acme_directory_uri = "https://acme-v02.api.letsencrypt.org/directory"
  renew_days         = 31
  dns_mapping        = { "example.com" = 1 }
}`,
			expect: `renew_days`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(tc.expect),
					},
				},
			})
		})
	}
}
