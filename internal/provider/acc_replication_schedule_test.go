package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// replicationTriggerConfig is a LOCAL push task between the two prerequisite
// datasets. trigger holds the attributes that decide what starts it.
func replicationTriggerConfig(name, trigger string) string {
	return fmt.Sprintf(`
resource "truenas_snapshot_task" "src" {
  dataset        = truenas_dataset.src.id
  recursive      = false
  lifetime_value = 2
  lifetime_unit  = "WEEK"
}

resource "truenas_replication" "test" {
  name             = %q
  direction        = "PUSH"
  transport        = "LOCAL"
  source_datasets  = [truenas_dataset.src.id]
  target_dataset   = truenas_dataset.dst.id
  recursive        = false
  enabled          = true
  retention_policy = "NONE"
%s}
`, name, trigger)
}

// TestAccReplicationResource_scheduleAndSnapshotTasks walks one push task
// through the three ways it can be started: on its own schedule, after a
// bound periodic snapshot task, and by hand. Each change is made in place,
// and each step must plan clean afterwards.
//
// The first step is also the proof that `auto = true` can be created at
// all. TrueNAS refuses an automatic task that has neither a schedule nor a
// bound snapshot task, and before these attributes existed the resource
// could set neither.
func TestAccReplicationResource_scheduleAndSnapshotTasks(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestrep%d", shortSuffix())
	src := randomName("acctestrepsrc")
	dst := randomName("acctestrepdst")
	prereqs := replicationPrereqs(src, dst)
	const res = "truenas_replication.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// On its own schedule. Only the hour is given: TrueNAS
				// fills the other fields, and they are read back.
				Config: prereqs + replicationTriggerConfig(name, `
  auto                       = true
  schedule                   = [{ hour = "*/6" }]
  also_include_naming_schema = ["auto-%Y-%m-%d_%H-%M"]
  hold_pending_snapshots     = true
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "auto", "true"),
					resource.TestCheckResourceAttr(res, "schedule.#", "1"),
					resource.TestCheckResourceAttr(res, "schedule.0.hour", "*/6"),
					resource.TestCheckResourceAttr(res, "schedule.0.minute", "00"),
					resource.TestCheckResourceAttr(res, "schedule.0.dow", "*"),
					resource.TestCheckResourceAttr(res, "schedule.0.begin", "00:00"),
					resource.TestCheckResourceAttr(res, "schedule.0.end", "23:59"),
					resource.TestCheckResourceAttr(res, "restrict_schedule.#", "0"),
					resource.TestCheckResourceAttr(res, "periodic_snapshot_tasks.#", "0"),
					resource.TestCheckResourceAttr(res, "hold_pending_snapshots", "true"),
					resource.TestCheckResourceAttr(res, "only_matching_schedule", "false"),
				),
			},
			{
				// After a bound snapshot task, limited to a daily window.
				// The schedule is removed with [], and the naming schema
				// by leaving it out: an update has to send both.
				Config: prereqs + replicationTriggerConfig(name, `
  auto                    = true
  schedule                = []
  periodic_snapshot_tasks = [tonumber(truenas_snapshot_task.src.id)]
  restrict_schedule       = [{ hour = "*/2", begin = "08:00", end = "20:00" }]
  hold_pending_snapshots  = true
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "schedule.#", "0"),
					resource.TestCheckResourceAttr(res, "periodic_snapshot_tasks.#", "1"),
					resource.TestCheckResourceAttrPair(res, "periodic_snapshot_tasks.0", "truenas_snapshot_task.src", "id"),
					resource.TestCheckResourceAttr(res, "restrict_schedule.#", "1"),
					resource.TestCheckResourceAttr(res, "restrict_schedule.0.hour", "*/2"),
					resource.TestCheckResourceAttr(res, "restrict_schedule.0.begin", "08:00"),
					resource.TestCheckResourceAttr(res, "restrict_schedule.0.end", "20:00"),
					resource.TestCheckNoResourceAttr(res, "also_include_naming_schema.#"),
				),
			},
			{
				// By hand. Everything that triggered it is cleared, and the
				// naming schema is changed in place.
				Config: prereqs + replicationTriggerConfig(name, `
  auto                       = false
  schedule                   = []
  periodic_snapshot_tasks    = []
  restrict_schedule          = []
  hold_pending_snapshots     = false
  also_include_naming_schema = ["manual-%Y-%m-%d_%H-%M"]
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "auto", "false"),
					resource.TestCheckResourceAttr(res, "schedule.#", "0"),
					resource.TestCheckResourceAttr(res, "restrict_schedule.#", "0"),
					resource.TestCheckResourceAttr(res, "periodic_snapshot_tasks.#", "0"),
					resource.TestCheckResourceAttr(res, "hold_pending_snapshots", "false"),
					resource.TestCheckResourceAttr(res, "also_include_naming_schema.0", "manual-%Y-%m-%d_%H-%M"),
				),
			},
			{
				ResourceName:      res,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccReplicationResource_autoNeedsATrigger pins the TrueNAS rule that the
// `auto` description and the changelog state: an automatic task with neither
// a schedule nor a bound periodic snapshot task is refused. If a TrueNAS
// version stops enforcing it, this fails and the wording has to change.
func TestAccReplicationResource_autoNeedsATrigger(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestrep%d", shortSuffix())
	src := randomName("acctestrepsrc")
	dst := randomName("acctestrepdst")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: replicationPrereqs(src, dst) + replicationTriggerConfig(name, `
  auto                       = true
  also_include_naming_schema = ["auto-%Y-%m-%d_%H-%M"]
`),
				// Terraform wraps the message, so any whitespace may
				// separate the words.
				ExpectError: regexp.MustCompile(`(?i)bound\s+to\s+a\s+periodic\s+snapshot\s+task\s+or\s+have\s+a\s+schedule`),
			},
		},
	})
}
