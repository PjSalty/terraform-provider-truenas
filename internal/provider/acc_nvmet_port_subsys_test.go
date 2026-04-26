package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// nvmetPortSubsysConfig builds an HCL config containing a port, a
// subsys, and the port_subsys join linking them.
func nvmetPortSubsysConfig(subsysName string, port int) string {
	return fmt.Sprintf(`
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = %d
}

resource "truenas_nvmet_subsys" "test" {
  name = %q
}

resource "truenas_nvmet_port_subsys" "test" {
  port_id   = tonumber(truenas_nvmet_port.test.id)
  subsys_id = tonumber(truenas_nvmet_subsys.test.id)
}
`, port, subsysName)
}

// nvmetPortSubsysConfigAlt swaps in a second subsys so _update can
// point the join at a different subsys_id (RequiresReplace).
func nvmetPortSubsysConfigAlt(subsysName, subsysNameAlt string, port int) string {
	return fmt.Sprintf(`
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = %d
}

resource "truenas_nvmet_subsys" "test" {
  name = %q
}

resource "truenas_nvmet_subsys" "alt" {
  name = %q
}

resource "truenas_nvmet_port_subsys" "test" {
  port_id   = tonumber(truenas_nvmet_port.test.id)
  subsys_id = tonumber(truenas_nvmet_subsys.alt.id)
}
`, port, subsysName, subsysNameAlt)
}

func TestAccNVMetPortSubsysResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	subsysName := fmt.Sprintf("accps%d", shortSuffix())
	port := nvmetPortPort()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetPortSubsysConfig(subsysName, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "port_id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "subsys_id"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_port_subsys.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccNVMetPortSubsysResource_update swaps subsys_id (RequiresReplace)
// to verify destroy/recreate behavior on the join row.
func TestAccNVMetPortSubsysResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	subsysName := fmt.Sprintf("accpsu%d", shortSuffix())
	subsysNameAlt := fmt.Sprintf("accpsua%d", shortSuffix())
	port := nvmetPortPort()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetPortSubsysConfig(subsysName, port),
				Check:  resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "id"),
			},
			{
				Config: nvmetPortSubsysConfigAlt(subsysName, subsysNameAlt, port),
				Check:  resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "id"),
			},
		},
	})
}

func TestAccNVMetPortSubsysResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	subsysName := fmt.Sprintf("accpsd%d", shortSuffix())
	port := nvmetPortPort()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetPortSubsysConfig(subsysName, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_nvmet_port_subsys.test"]
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
						return c.DeleteNVMetPortSubsys(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
