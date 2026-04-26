package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccISCSIPortalResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_portal" "test" {
  comment = "acctest"
  listen = [
    {
      ip = "0.0.0.0"
    },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "comment", "acctest"),
					resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "listen.#", "1"),
					resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "listen.0.ip", "0.0.0.0"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_portal.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_portal.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccISCSIPortalResource_update flips the comment on the portal in a
// second apply.
func TestAccISCSIPortalResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_portal" "test" {
  comment = "initial"
  listen = [
    { ip = "0.0.0.0" },
  ]
}
`,
				Check: resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "comment", "initial"),
			},
			{
				Config: `
resource "truenas_iscsi_portal" "test" {
  comment = "updated"
  listen = [
    { ip = "0.0.0.0" },
  ]
}
`,
				Check: resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "comment", "updated"),
			},
		},
	})
}

// TestAccISCSIPortalResource_disappears deletes the portal out-of-band to
// verify drift recovery.
func TestAccISCSIPortalResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_portal" "test" {
  comment = "acctest-disappears"
  listen = [
    { ip = "0.0.0.0" },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_portal.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_iscsi_portal.test"]
						if !ok {
							return fmt.Errorf("resource truenas_iscsi_portal.test not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("bad id %q: %w", rs.Primary.ID, err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteISCSIPortal(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
