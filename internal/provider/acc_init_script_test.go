package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccInitScriptResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_init_script" "test" {
  type    = "COMMAND"
  command = "/bin/true"
  when    = "POSTINIT"
  enabled = false
  comment = "acctest"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_init_script.test", "type", "COMMAND"),
					resource.TestCheckResourceAttr("truenas_init_script.test", "command", "/bin/true"),
					resource.TestCheckResourceAttr("truenas_init_script.test", "when", "POSTINIT"),
					resource.TestCheckResourceAttrSet("truenas_init_script.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_init_script.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccInitScriptResource_update changes the comment in place.
func TestAccInitScriptResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	cfg := func(comment string) string {
		return fmt.Sprintf(`
resource "truenas_init_script" "test" {
  type    = "COMMAND"
  command = "/bin/true"
  when    = "POSTINIT"
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
				Check:  resource.TestCheckResourceAttr("truenas_init_script.test", "comment", "acctest initial"),
			},
			{
				Config: cfg("acctest updated"),
				Check:  resource.TestCheckResourceAttr("truenas_init_script.test", "comment", "acctest updated"),
			},
		},
	})
}

// TestAccInitScriptResource_disappears deletes the init script out-of-band.
func TestAccInitScriptResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_init_script" "test" {
  type    = "COMMAND"
  command = "/bin/true"
  when    = "POSTINIT"
  enabled = false
  comment = "acctest disappears"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_init_script.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_init_script.test"]
						if !ok {
							return fmt.Errorf("truenas_init_script.test not in state")
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
						return c.DeleteInitScript(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
