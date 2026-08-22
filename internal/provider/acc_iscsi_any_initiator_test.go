package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccISCSITargetResource_anyInitiator creates a target group that names a
// portal and no initiator group, which is upstream's default and means "allow
// any initiator". The provider marked initiator Required, so this whole shape
// was unreachable.
func TestAccISCSITargetResource_anyInitiator(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	target := fmt.Sprintf("iqn.2026-01.com.example:any%d", shortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_iscsi_portal" "any" {
  comment = "acc any-initiator"

  listen = [
    {
      ip   = "0.0.0.0"
      port = 3260
    },
  ]
}

resource "truenas_iscsi_target" "any" {
  name = %q
  mode = "ISCSI"

  groups = [
    {
      portal = tonumber(truenas_iscsi_portal.any.id)
    },
  ]
}
`, target),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_target.any", "name", target),
					resource.TestCheckResourceAttr("truenas_iscsi_target.any", "groups.#", "1"),
				),
			},
		},
	})
}
