package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNVMetSubsysResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestsubsys%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_subsys" "test" {
  name = %q
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_subsys.test", "name", name),
					resource.TestCheckResourceAttrSet("truenas_nvmet_subsys.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_subsys.test", "subnqn"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_subsys.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccNVMetSubsysResource_update toggles allow_any_host to verify
// the provider issues a PUT rather than a destroy/recreate.
func TestAccNVMetSubsysResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestsubsys%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_subsys" "test" {
  name           = %q
  allow_any_host = false
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_nvmet_subsys.test", "allow_any_host", "false"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_subsys" "test" {
  name           = %q
  allow_any_host = true
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_nvmet_subsys.test", "allow_any_host", "true"),
			},
		},
	})
}

// TestAccNVMetSubsysResource_disappears deletes the subsystem out of
// band and confirms Terraform detects the drift (non-empty plan).
func TestAccNVMetSubsysResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestsubsys%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_subsys" "test" {
  name = %q
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_subsys.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_nvmet_subsys.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parsing id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteNVMetSubsys(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
