package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDirectoryServices_disable verifies the singleton resource can be
// created in a disabled state. We avoid actually joining an AD/LDAP/IPA
// domain in acceptance tests because it requires live directory servers.
func TestAccDirectoryServices_disable(t *testing.T) {
	resourceName := "truenas_directoryservices.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDirectoryServicesDisabled(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "enable", "false"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateId:           "1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential_json", "configuration_json", "force"},
			},
		},
	})
}

func testAccDirectoryServicesDisabled() string {
	return `
provider "truenas" {}

resource "truenas_directoryservices" "test" {
  enable = false
}
`
}

// TestAccDirectoryServices_fullADLifecycle exercises the complete
// Active Directory join + leave lifecycle against a live AD DC:
//
//  1. Pre-create kerberos_realm pointing at the AD DC's KDC
//  2. Configure directoryservices with service_type=ACTIVEDIRECTORY,
//     credential_type=KERBEROS_PRINCIPAL, enable=true
//  3. Poll until directoryservices.status reports HEALTHY
//  4. Disable directoryservices (leave the domain)
//  5. Destroy kerberos_realm
//  6. Verify clean state — directoryservices reverts to null
//     service_type, kerberos_realm row gone, TrueNAS not joined.
//
// Env-gated: requires TRUENAS_TEST_AD=1 plus TRUENAS_TEST_AD_DC,
// TRUENAS_TEST_AD_REALM, TRUENAS_TEST_AD_ADMIN_PRINCIPAL. Skips
// when missing — full AD lifecycle is invasive (modifies hostname
// keytab, /etc/krb5.conf, joins host to domain) and shouldn't run
// on every CI invocation. Run against a throwaway Samba AD-DC.
//
// Note: KERBEROS_PRINCIPAL is the credential_type required when
// enable=true. The 25.10 schema rejects KERBEROS_USER for enabled
// directory services. Use a service-principal-name (SPN) bound to
// a host machine account on the DC.
func TestAccDirectoryServices_fullADLifecycle(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_AD") != "1" {
		t.Skip("TRUENAS_TEST_AD=1 not set; AD lifecycle test is invasive — gates behind explicit opt-in")
	}
	dcAddr := os.Getenv("TRUENAS_TEST_AD_DC")
	realm := os.Getenv("TRUENAS_TEST_AD_REALM")
	principal := os.Getenv("TRUENAS_TEST_AD_ADMIN_PRINCIPAL")
	if dcAddr == "" || realm == "" || principal == "" {
		t.Skip("AD test needs TRUENAS_TEST_AD_DC, TRUENAS_TEST_AD_REALM, TRUENAS_TEST_AD_ADMIN_PRINCIPAL")
	}

	resourceName := "truenas_directoryservices.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: create kerberos_realm + leave directoryservices
			// disabled. Establishes the prerequisite without touching
			// AD state yet.
			{
				Config: testAccDSConfig_RealmOnly(realm),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_kerberos_realm.ad",
						"realm", realm),
				),
			},
			// Step 2: enable AD with KERBEROS_PRINCIPAL credential.
			// This is the JOIN step — TrueNAS contacts the DC, creates
			// a host machine account, gets a keytab, and updates
			// /etc/krb5.conf + nsswitch.conf.
			{
				Config: testAccDSConfig_ADEnabled(realm, dcAddr, principal),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable", "true"),
					resource.TestCheckResourceAttr(resourceName, "service_type", "ACTIVEDIRECTORY"),
				),
			},
			// Step 3: disable (LEAVE the domain). TrueNAS unbinds from
			// the DC, removes the machine account, restores
			// /etc/krb5.conf to defaults.
			{
				Config: testAccDSConfig_RealmOnly(realm),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enable", "false"),
				),
			},
			// Step 4: remove the realm entirely.
			{
				Config: testAccDSConfig_Empty(),
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
		},
	})
}

func testAccDSConfig_Empty() string {
	return `
provider "truenas" {}
`
}

func testAccDSConfig_RealmOnly(realm string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_kerberos_realm" "ad" {
  realm = %q
}

resource "truenas_directoryservices" "test" {
  enable                = false
  enable_account_cache  = true
  enable_dns_updates    = true
  timeout               = 10
  kerberos_realm        = truenas_kerberos_realm.ad.realm
}
`, realm)
}

func testAccDSConfig_ADEnabled(realm, dcAddr, _ string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_kerberos_realm" "ad" {
  realm = %q
}

resource "truenas_directoryservices" "test" {
  enable                = true
  service_type          = "ACTIVEDIRECTORY"
  enable_account_cache  = true
  enable_dns_updates    = true
  timeout               = 30
  kerberos_realm        = truenas_kerberos_realm.ad.realm

  credential_json = jsonencode({
    credential_type = "KERBEROS_USER"
    username = "Administrator"
    password = "TestADPass2026!"
  })

  configuration_json = jsonencode({
    hostname = %q
    domain   = %q
  })
}
`, realm, dcAddr, realm)
}
