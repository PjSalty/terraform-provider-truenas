package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccNameRulesResource_dottedAndMixedCase creates the names the provider
// used to refuse. Relaxing a validator is an assertion about what the server
// accepts, and an assertion nothing applies is a guess.
func TestAccNameRulesResource_dottedAndMixedCase(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	// Dot and mixed case in both, which the old patterns rejected outright.
	group := randomName("Acc.Grp")
	user := randomName("Acc.Usr")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_group" "dotted" {
  name = %q
}

resource "truenas_user" "dotted" {
  username          = %q
  full_name         = "Dotted Mixed Case"
  group             = tonumber(truenas_group.dotted.id)
  password_disabled = true
  home              = "/var/empty"
  shell             = "/usr/sbin/nologin"
}
`, group, user),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_group.dotted", "name", group),
					resource.TestCheckResourceAttr("truenas_user.dotted", "username", user),
				),
			},
		},
	})
}
