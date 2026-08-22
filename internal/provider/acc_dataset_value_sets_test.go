package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccDatasetResource_modernValueSets applies values the provider used to
// reject outright: a graded ZSTD level and a recordsize above 1M.
//
// Widening a OneOf from the upstream models is an assertion about what the
// server accepts, and an assertion nothing applies is a guess. This applies
// them.
func TestAccDatasetResource_modernValueSets(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestvalsets")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_dataset" "vals" {
  pool        = "test"
  name        = %q
  compression = "ZSTD-19"
  record_size = "4M"
}
`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.vals", "compression", "ZSTD-19"),
					resource.TestCheckResourceAttr("truenas_dataset.vals", "record_size", "4M"),
				),
			},
			{
				// A different graded level must be an in-place update, and the
				// smallest recordsize the server lists must be accepted too.
				Config: fmt.Sprintf(`
resource "truenas_dataset" "vals" {
  pool        = "test"
  name        = %q
  compression = "ZSTD-FAST-10"
  record_size = "4M"
}
`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_dataset.vals", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_dataset.vals", "compression", "ZSTD-FAST-10"),
			},
		},
	})
}
