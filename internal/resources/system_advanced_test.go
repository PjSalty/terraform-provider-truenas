package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Delete on this singleton is a no-op, so whatever the last applied step
// sets is what the test box keeps. These tests change only the message of
// the day, the login banner, the syslog settings and the boot scrub
// interval: none of them can cut off the network or the API, or stop the
// box booting. The serial console, kernel options, kdump, the debug kernel
// and the console menu are left alone on purpose, since they act on the
// boot menu and the local console. Remote servers use the documentation
// address 192.0.2.10 over UDP, which sends datagrams nowhere.
//
// Every test ends on the values a fresh TrueNAS 25.10 has.
const (
	systemAdvancedDefaultMotd  = "Welcome to TrueNAS"
	systemAdvancedDefaultLevel = "F_INFO"
	systemAdvancedDefaultScrub = 7
)

func testAccSystemAdvanced(motd, banner, level string, fqdn bool, scrub int, servers string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_system_advanced" "test" {
  motd          = %q
  login_banner  = %q
  sysloglevel   = %q
  fqdn_syslog   = %t
  boot_scrub    = %d
  syslogservers = %s
}
`, motd, banner, level, fqdn, scrub, servers)
}

func testAccSystemAdvancedDefaults() string {
	return testAccSystemAdvanced(systemAdvancedDefaultMotd, "", systemAdvancedDefaultLevel, false, systemAdvancedDefaultScrub, "[]")
}

func TestAccSystemAdvanced_basic(t *testing.T) {
	resourceName := "truenas_system_advanced.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemAdvancedDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "system_advanced"),
					resource.TestCheckResourceAttr(resourceName, "motd", systemAdvancedDefaultMotd),
					resource.TestCheckResourceAttr(resourceName, "sysloglevel", systemAdvancedDefaultLevel),
					resource.TestCheckResourceAttr(resourceName, "boot_scrub", "7"),
					resource.TestCheckResourceAttr(resourceName, "syslogservers.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "system_advanced",
				ImportStateVerify: true,
			},
		},
	})
}

// Every managed setting here changes in place and reads back as written,
// and the last step puts the fresh-install values back.
func TestAccSystemAdvanced_update(t *testing.T) {
	resourceName := "truenas_system_advanced.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemAdvancedDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: testAccSystemAdvanced("Managed by Terraform", "Authorised use only", "F_NOTICE", true, 14, "[]"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "motd", "Managed by Terraform"),
					resource.TestCheckResourceAttr(resourceName, "login_banner", "Authorised use only"),
					resource.TestCheckResourceAttr(resourceName, "sysloglevel", "F_NOTICE"),
					resource.TestCheckResourceAttr(resourceName, "fqdn_syslog", "true"),
					resource.TestCheckResourceAttr(resourceName, "boot_scrub", "14"),
				),
			},
			{
				Config: testAccSystemAdvancedDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "motd", systemAdvancedDefaultMotd),
					resource.TestCheckResourceAttr(resourceName, "login_banner", ""),
					resource.TestCheckResourceAttr(resourceName, "sysloglevel", systemAdvancedDefaultLevel),
					resource.TestCheckResourceAttr(resourceName, "fqdn_syslog", "false"),
					resource.TestCheckResourceAttr(resourceName, "boot_scrub", "7"),
				),
			},
		},
	})
}

// Remote syslog: two UDP servers, one with a port, then one, then none. The
// order is kept as written, the default transport is UDP, and an empty list
// removes every server.
func TestAccSystemAdvanced_syslogServers(t *testing.T) {
	resourceName := "truenas_system_advanced.test"
	two := `[
    { host = "192.0.2.10" },
    { host = "192.0.2.11:5514", transport = "UDP" },
  ]`
	reversed := `[
    { host = "192.0.2.11:5514", transport = "UDP" },
    { host = "192.0.2.10" },
  ]`
	one := `[{ host = "192.0.2.10" }]`

	step := func(servers string, count string, first string) resource.TestStep {
		return resource.TestStep{
			Config: testAccSystemAdvanced(systemAdvancedDefaultMotd, "", systemAdvancedDefaultLevel, false, systemAdvancedDefaultScrub, servers),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "syslogservers.#", count),
				resource.TestCheckResourceAttr(resourceName, "syslogservers.0.host", first),
				resource.TestCheckResourceAttr(resourceName, "syslogservers.0.transport", "UDP"),
				resource.TestCheckNoResourceAttr(resourceName, "syslogservers.0.tls_certificate"),
			),
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			step(two, "2", "192.0.2.10"),
			step(reversed, "2", "192.0.2.11:5514"),
			step(one, "1", "192.0.2.10"),
			{
				Config: testAccSystemAdvancedDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "syslogservers.#", "0"),
			},
		},
	})
}

// A configuration that sets one attribute leaves the rest as TrueNAS has
// them and still plans nothing afterwards.
func TestAccSystemAdvanced_partial(t *testing.T) {
	resourceName := "truenas_system_advanced.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemAdvancedDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: `
provider "truenas" {}

resource "truenas_system_advanced" "test" {
  motd = "Only the motd"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "motd", "Only the motd"),
					resource.TestCheckResourceAttr(resourceName, "boot_scrub", "7"),
					resource.TestCheckResourceAttr(resourceName, "sysloglevel", systemAdvancedDefaultLevel),
				),
			},
			{
				Config: testAccSystemAdvancedDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "motd", systemAdvancedDefaultMotd),
			},
		},
	})
}

// Values TrueNAS would misread or reject fail at plan time, before anything
// is written.
func TestAccSystemAdvanced_invalidRejected(t *testing.T) {
	cases := map[string]struct {
		servers string
		pattern string
	}{
		"bare IPv6":   {`[{ host = "2001:db8::10" }]`, `must be in brackets`},
		"certificate": {`[{ host = "192.0.2.10", tls_certificate = 1 }]`, `only be set when transport`},
		"duplicate":   {`[{ host = "192.0.2.10" }, { host = "192.0.2.10", transport = "TCP" }]`, `listed more than once`},
		"three":       {`[{ host = "192.0.2.10" }, { host = "192.0.2.11" }, { host = "192.0.2.12" }]`, `at most 2`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      testAccSystemAdvanced(systemAdvancedDefaultMotd, "", systemAdvancedDefaultLevel, false, systemAdvancedDefaultScrub, tc.servers),
						ExpectError: regexp.MustCompile(tc.pattern),
						PlanOnly:    true,
					},
				},
			})
		})
	}
}
