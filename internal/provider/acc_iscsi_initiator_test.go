package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccISCSIInitiatorResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_initiator" "test" {
  comment    = "acctest"
  initiators = []
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "comment", "acctest"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_initiator.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_initiator.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccISCSIInitiatorResource_update flips the comment and extends the
// initiators list.
func TestAccISCSIInitiatorResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_initiator" "test" {
  comment    = "initial"
  initiators = []
}
`,
				Check: resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "comment", "initial"),
			},
			{
				Config: `
resource "truenas_iscsi_initiator" "test" {
  comment    = "updated"
  initiators = ["iqn.2026-04.com.example:test"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "comment", "updated"),
					resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "initiators.#", "1"),
				),
			},
		},
	})
}

// TestAccISCSIInitiatorResource_disappears deletes the initiator group
// out-of-band.
func TestAccISCSIInitiatorResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_initiator" "test" {
  comment    = "acctest-disappears"
  initiators = []
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_initiator.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_iscsi_initiator.test"]
						if !ok {
							return fmt.Errorf("resource truenas_iscsi_initiator.test not found in state")
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
						return c.DeleteISCSIInitiator(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
