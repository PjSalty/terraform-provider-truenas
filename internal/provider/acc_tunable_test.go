package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccTunableResource_basic uses kernel.acct (BSD process accounting),
// which is safe to set to 0 on SCALE: it does nothing until enabled and
// has no side effects. We also set enabled=false so the sysctl is never
// applied at all.
func TestAccTunableResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_tunable" "test" {
  type    = "SYSCTL"
  var     = "kernel.acct"
  value   = "0"
  enabled = false
  comment = "acctest"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_tunable.test", "type", "SYSCTL"),
					resource.TestCheckResourceAttr("truenas_tunable.test", "var", "kernel.acct"),
					resource.TestCheckResourceAttr("truenas_tunable.test", "value", "0"),
					resource.TestCheckResourceAttr("truenas_tunable.test", "enabled", "false"),
					resource.TestCheckResourceAttrSet("truenas_tunable.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_tunable.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccTunableResource_update changes the comment in place. The var
// and type fields have RequiresReplace and are held constant.
func TestAccTunableResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	cfg := func(comment string) string {
		return fmt.Sprintf(`
resource "truenas_tunable" "test" {
  type    = "SYSCTL"
  var     = "kernel.acct"
  value   = "0"
  enabled = false
  comment = %q
}
`, comment)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("acctest initial"),
				Check:  resource.TestCheckResourceAttr("truenas_tunable.test", "comment", "acctest initial"),
			},
			{
				Config: cfg("acctest updated"),
				Check:  resource.TestCheckResourceAttr("truenas_tunable.test", "comment", "acctest updated"),
			},
		},
	})
}

// TestAccTunableResource_disappears deletes the tunable out-of-band.
func TestAccTunableResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_tunable" "test" {
  type    = "SYSCTL"
  var     = "kernel.acct"
  value   = "0"
  enabled = false
  comment = "acctest disappears"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_tunable.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_tunable.test"]
						if !ok {
							return fmt.Errorf("truenas_tunable.test not in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parse id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteTunable(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
