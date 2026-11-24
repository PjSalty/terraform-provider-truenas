package resources_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// What a fresh TrueNAS has: the window is off, 18:00 to 09:00, every day.
// Delete on this singleton is a no-op, so whatever the last applied step
// sets is what the test box keeps. Every test therefore ends on these.
const (
	resilverDefaultBegin = "18:00"
	resilverDefaultEnd   = "09:00"
)

var resilverEveryDay = []int{1, 2, 3, 4, 5, 6, 7}

func testAccResilverConfig(enabled bool, begin, end string, days []int) string {
	list := make([]string, 0, len(days))
	for _, day := range days {
		list = append(list, fmt.Sprint(day))
	}
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_resilver_config" "test" {
  enabled = %t
  begin   = %q
  end     = %q
  weekday = [%s]
}
`, enabled, begin, end, strings.Join(list, ", "))
}

func testAccResilverConfigDefaults() string {
	return testAccResilverConfig(false, resilverDefaultBegin, resilverDefaultEnd, resilverEveryDay)
}

func TestAccResilverConfig_basic(t *testing.T) {
	resourceName := "truenas_resilver_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResilverConfig(true, resilverDefaultBegin, resilverDefaultEnd, resilverEveryDay),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "resilver_config"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "begin", resilverDefaultBegin),
					resource.TestCheckResourceAttr(resourceName, "end", resilverDefaultEnd),
					resource.TestCheckResourceAttr(resourceName, "weekday.#", "7"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "resilver_config",
				ImportStateVerify: true,
			},
			{
				Config: testAccResilverConfigDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
			},
		},
	})
}

// Every attribute changes in place, and a window that rolls over midnight
// on two days reads back exactly as written. The days are given out of
// order on purpose: weekday is a set, and TrueNAS stores the days in the
// order they were sent, so a list would show a diff here.
func TestAccResilverConfig_update(t *testing.T) {
	resourceName := "truenas_resilver_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResilverConfigDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
			},
			{
				Config: testAccResilverConfig(true, "22:30", "06:00", []int{7, 6}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "begin", "22:30"),
					resource.TestCheckResourceAttr(resourceName, "end", "06:00"),
					resource.TestCheckResourceAttr(resourceName, "weekday.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "weekday.*", "6"),
					resource.TestCheckTypeSetElemAttr(resourceName, "weekday.*", "7"),
				),
			},
			{
				Config: testAccResilverConfigDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "begin", resilverDefaultBegin),
					resource.TestCheckResourceAttr(resourceName, "weekday.#", "7"),
				),
			},
		},
	})
}

// A configuration that sets only enabled must leave the hours and days as
// TrueNAS has them, and still plan nothing afterwards.
func TestAccResilverConfig_partial(t *testing.T) {
	resourceName := "truenas_resilver_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResilverConfigDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: `
provider "truenas" {}

resource "truenas_resilver_config" "test" {
  enabled = true
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "begin", resilverDefaultBegin),
					resource.TestCheckResourceAttr(resourceName, "end", resilverDefaultEnd),
					resource.TestCheckResourceAttr(resourceName, "weekday.#", "7"),
				),
			},
			{
				Config: testAccResilverConfigDefaults(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
			},
		},
	})
}

// TrueNAS accepts "9:00" and reads it back as "09:00". That has to fail at
// plan time, before anything is written, or the plan never settles.
func TestAccResilverConfig_unpaddedTimeRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResilverConfig(true, resilverDefaultBegin, "9:00", resilverEveryDay),
				ExpectError: regexp.MustCompile(`zero-padded 24 hour time`),
				PlanOnly:    true,
			},
		},
	})
}
