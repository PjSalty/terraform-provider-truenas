package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// What a fresh TrueNAS has: the stock basename and no pool threshold (null
// on the server, 0 here). Delete on this singleton is a no-op, so whatever
// the last applied step sets is what the test box keeps, and every test
// ends on these. The tests never touch listen_port, alua, iser or the iSNS
// servers: a changed port or an unreachable iSNS server could cut off or
// block the box, and alua and iser are read-only.
const (
	iscsiGlobalDefaultBasename  = "iqn.2005-10.org.freenas.ctl"
	iscsiGlobalDefaultThreshold = 0
)

func testAccISCSIGlobal(basename string, threshold int) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_iscsi_global" "test" {
  basename             = %q
  pool_avail_threshold = %d
}
`, basename, threshold)
}

func testAccISCSIGlobalDefaults() string {
	return testAccISCSIGlobal(iscsiGlobalDefaultBasename, iscsiGlobalDefaultThreshold)
}

func TestAccISCSIGlobal_basic(t *testing.T) {
	resourceName := "truenas_iscsi_global.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIGlobalDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "iscsi_global"),
					resource.TestCheckResourceAttr(resourceName, "basename", iscsiGlobalDefaultBasename),
					resource.TestCheckResourceAttr(resourceName, "pool_avail_threshold", "0"),
					resource.TestCheckResourceAttr(resourceName, "listen_port", "3260"),
					resource.TestCheckResourceAttr(resourceName, "isns_servers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "alua", "false"),
					resource.TestCheckResourceAttr(resourceName, "iser", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "iscsi_global",
				ImportStateVerify: true,
			},
		},
	})
}

// Both settable attributes change in place, the threshold is set and then
// removed again through 0 (null on the server), and the plan settles after
// each step. The last step puts the fresh-install values back.
func TestAccISCSIGlobal_update(t *testing.T) {
	resourceName := "truenas_iscsi_global.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIGlobalDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: testAccISCSIGlobal("iqn.2024-01.test.acc:ctl", 80),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "basename", "iqn.2024-01.test.acc:ctl"),
					resource.TestCheckResourceAttr(resourceName, "pool_avail_threshold", "80"),
				),
			},
			{
				Config: testAccISCSIGlobal("iqn.2024-01.test.acc:ctl", 0),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "pool_avail_threshold", "0"),
			},
			{
				Config: testAccISCSIGlobalDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "basename", iscsiGlobalDefaultBasename),
					resource.TestCheckResourceAttr(resourceName, "pool_avail_threshold", "0"),
				),
			},
		},
	})
}

// A configuration that sets only the threshold must leave the basename as
// TrueNAS has it, and the iSNS servers too: iscsi.global.update would empty
// them if the provider did not send them back.
func TestAccISCSIGlobal_partial(t *testing.T) {
	resourceName := "truenas_iscsi_global.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIGlobalDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: `
provider "truenas" {}

resource "truenas_iscsi_global" "test" {
  pool_avail_threshold = 70
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pool_avail_threshold", "70"),
					resource.TestCheckResourceAttr(resourceName, "basename", iscsiGlobalDefaultBasename),
					resource.TestCheckResourceAttr(resourceName, "listen_port", "3260"),
				),
			},
			{
				Config: testAccISCSIGlobalDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "pool_avail_threshold", "0"),
			},
		},
	})
}

// Values TrueNAS would refuse have to fail at plan time, before anything
// is written.
func TestAccISCSIGlobal_invalidRejected(t *testing.T) {
	cfg := func(attrs string) string {
		return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_iscsi_global" "test" {
  %s
}
`, attrs)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cfg("pool_avail_threshold = 100"),
				ExpectError: regexp.MustCompile(`(?s)pool_avail_threshold.*between 0 and 99`),
				PlanOnly:    true,
			},
			{
				Config:      cfg("listen_port = 80"),
				ExpectError: regexp.MustCompile(`(?s)listen_port.*between 1025 and 65535`),
				PlanOnly:    true,
			},
		},
	})
}
