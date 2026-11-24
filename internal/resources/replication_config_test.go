package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// What a fresh TrueNAS has: five replication tasks at a time. Delete on this
// singleton is a no-op, so whatever the last applied step sets is what the
// test box keeps, and every test ends on this. The limit only decides how
// many replication tasks run together, so no value of it can cut off the
// box's network or API.
const replicationConfigDefaultLimit = 5

func testAccReplicationConfig(limit int) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_replication_config" "test" {
  max_parallel_replication_tasks = %d
}
`, limit)
}

func TestAccReplicationConfig_basic(t *testing.T) {
	resourceName := "truenas_replication_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfig(replicationConfigDefaultLimit),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "replication_config"),
					resource.TestCheckResourceAttr(resourceName, "max_parallel_replication_tasks", "5"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "replication_config",
				ImportStateVerify: true,
			},
		},
	})
}

// A limit is set, removed through 0 (null on the server) and set again, and
// the plan settles after each step. The last step puts the fresh-install
// value back.
func TestAccReplicationConfig_update(t *testing.T) {
	resourceName := "truenas_replication_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfig(replicationConfigDefaultLimit),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: testAccReplicationConfig(2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "max_parallel_replication_tasks", "2"),
			},
			{
				Config: testAccReplicationConfig(0),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "max_parallel_replication_tasks", "0"),
			},
			{
				Config: testAccReplicationConfig(replicationConfigDefaultLimit),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "max_parallel_replication_tasks", "5"),
			},
		},
	})
}

// A configuration that leaves the limit out keeps what TrueNAS has and
// plans nothing, including when that is "no limit".
func TestAccReplicationConfig_unset(t *testing.T) {
	resourceName := "truenas_replication_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfig(replicationConfigDefaultLimit),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: `
provider "truenas" {}

resource "truenas_replication_config" "test" {}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "max_parallel_replication_tasks", "5"),
			},
		},
	})
}

// A negative limit has to fail at plan time, before anything is written.
func TestAccReplicationConfig_negativeRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccReplicationConfig(-1),
				ExpectError: regexp.MustCompile(`(?s)max_parallel_replication_tasks.*at least 0`),
				PlanOnly:    true,
			},
		},
	})
}
