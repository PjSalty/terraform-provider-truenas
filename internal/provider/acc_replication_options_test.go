package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// replicationOptionsConfig is a manual LOCAL push task between the two
// prerequisite datasets, with the given retention policy and options.
func replicationOptionsConfig(name, retention, options string) string {
	return fmt.Sprintf(`
resource "truenas_replication" "test" {
  name             = %q
  direction        = "PUSH"
  transport        = "LOCAL"
  source_datasets  = [truenas_dataset.src.id]
  target_dataset   = truenas_dataset.dst.id
  recursive        = false
  auto             = false
  enabled          = true
  retention_policy = %q
%s}
`, name, retention, options)
}

// TestAccReplicationResource_options sets the transfer options on a task,
// clears the ones that can be cleared, and then gives it a custom retention
// with its own lifetimes. Every change is made in place, and every step
// must plan clean afterwards.
//
// compression, speed_limit and sudo need an SSH transport and are covered by
// TestAccReplicationResource_sshOneApply. exclude needs a child dataset and
// is covered by the unit tests only.
func TestAccReplicationResource_options(t *testing.T) {
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
				Config: prereqs + replicationOptionsConfig(name, "NONE", `
  also_include_naming_schema = ["auto-%Y-%m-%d_%H-%M"]
  readonly                   = "IGNORE"
  allow_from_scratch         = true
  retries                    = 3
  properties_exclude         = ["mountpoint"]
  properties_override        = { atime = "off" }
  large_block                = false
  embed                      = true
  compressed                 = false
  logging_level              = "DEBUG"
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "readonly", "IGNORE"),
					resource.TestCheckResourceAttr(res, "allow_from_scratch", "true"),
					resource.TestCheckResourceAttr(res, "retries", "3"),
					resource.TestCheckResourceAttr(res, "properties_exclude.#", "1"),
					resource.TestCheckResourceAttr(res, "properties_override.atime", "off"),
					resource.TestCheckResourceAttr(res, "large_block", "false"),
					resource.TestCheckResourceAttr(res, "embed", "true"),
					resource.TestCheckResourceAttr(res, "compressed", "false"),
					resource.TestCheckResourceAttr(res, "logging_level", "DEBUG"),
					// Left out, so read back: the TrueNAS defaults, and the
					// empty value for what TrueNAS holds as null.
					resource.TestCheckResourceAttr(res, "properties", "true"),
					resource.TestCheckResourceAttr(res, "replicate", "false"),
					resource.TestCheckResourceAttr(res, "sudo", "false"),
					resource.TestCheckResourceAttr(res, "compression", ""),
					resource.TestCheckResourceAttr(res, "speed_limit", "0"),
					resource.TestCheckResourceAttr(res, "name_regex", ""),
					resource.TestCheckResourceAttr(res, "exclude.#", "0"),
					resource.TestCheckResourceAttr(res, "lifetimes.#", "0"),
				),
			},
			{
				// Clear what can be cleared, and select snapshots by a
				// regular expression in place of the naming schema.
				Config: prereqs + replicationOptionsConfig(name, "NONE", `
  name_regex          = "auto-.*"
  readonly            = "SET"
  properties_exclude  = []
  properties_override = {}
  logging_level       = ""
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "name_regex", "auto-.*"),
					resource.TestCheckResourceAttr(res, "readonly", "SET"),
					resource.TestCheckResourceAttr(res, "properties_exclude.#", "0"),
					resource.TestCheckResourceAttr(res, "properties_override.%", "0"),
					resource.TestCheckResourceAttr(res, "logging_level", ""),
					resource.TestCheckNoResourceAttr(res, "also_include_naming_schema.#"),
					// Not mentioned in this step, so kept from the first.
					resource.TestCheckResourceAttr(res, "retries", "3"),
					resource.TestCheckResourceAttr(res, "embed", "true"),
				),
			},
			{
				// A custom retention with one extra lifetime. TrueNAS does
				// not accept a regular expression with it.
				Config: prereqs + replicationOptionsConfig(name, "CUSTOM", `
  also_include_naming_schema = ["auto-%Y-%m-%d_%H-%M"]
  name_regex                 = ""
  lifetime_value             = 2
  lifetime_unit              = "WEEK"
  lifetimes = [{
    schedule       = { minute = "0", hour = "0", dom = "*", month = "*", dow = "7" }
    lifetime_value = 3
    lifetime_unit  = "MONTH"
  }]
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "retention_policy", "CUSTOM"),
					resource.TestCheckResourceAttr(res, "name_regex", ""),
					resource.TestCheckResourceAttr(res, "lifetimes.#", "1"),
					resource.TestCheckResourceAttr(res, "lifetimes.0.schedule.dow", "7"),
					resource.TestCheckResourceAttr(res, "lifetimes.0.lifetime_value", "3"),
					resource.TestCheckResourceAttr(res, "lifetimes.0.lifetime_unit", "MONTH"),
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
