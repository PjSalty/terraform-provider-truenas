package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// nvmetHostSubsysConfig builds an HCL config containing a host, a
// subsystem, and the host_subsys join linking them.
func nvmetHostSubsysConfig(hostnqn, subsysName string) string {
	return fmt.Sprintf(`
resource "truenas_nvmet_host" "test" {
  hostnqn = %q
}

resource "truenas_nvmet_subsys" "test" {
  name = %q
}

resource "truenas_nvmet_host_subsys" "test" {
  host_id   = tonumber(truenas_nvmet_host.test.id)
  subsys_id = tonumber(truenas_nvmet_subsys.test.id)
}
`, hostnqn, subsysName)
}

// nvmetHostSubsysConfigAlt is the same as nvmetHostSubsysConfig but
// creates a second host so the _update step can point host_subsys at a
// different host_id (which triggers RequiresReplace).
func nvmetHostSubsysConfigAlt(hostnqn, hostnqnAlt, subsysName string) string {
	return fmt.Sprintf(`
resource "truenas_nvmet_host" "test" {
  hostnqn = %q
}

resource "truenas_nvmet_host" "alt" {
  hostnqn = %q
}

resource "truenas_nvmet_subsys" "test" {
  name = %q
}

resource "truenas_nvmet_host_subsys" "test" {
  host_id   = tonumber(truenas_nvmet_host.alt.id)
  subsys_id = tonumber(truenas_nvmet_subsys.test.id)
}
`, hostnqn, hostnqnAlt, subsysName)
}

func TestAccNVMetHostSubsysResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	hostnqn := fmt.Sprintf("nqn.2014-08.org.nvmexpress:uuid:%s", randomUUID(t))
	subsysName := fmt.Sprintf("acchs%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetHostSubsysConfig(hostnqn, subsysName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "host_id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "subsys_id"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_host_subsys.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccNVMetHostSubsysResource_update swaps host_id (RequiresReplace
// attribute) to verify the provider correctly destroys and recreates
// the join row. Both host_id and subsys_id are the only attributes and
// both require replacement, so this is the closest to an "update"
// round-trip meaningful for this resource.
func TestAccNVMetHostSubsysResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	hostnqn := fmt.Sprintf("nqn.2014-08.org.nvmexpress:uuid:%s", randomUUID(t))
	hostnqnAlt := fmt.Sprintf("nqn.2014-08.org.nvmexpress:uuid:%s", randomUUID(t))
	subsysName := fmt.Sprintf("acchsu%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetHostSubsysConfig(hostnqn, subsysName),
				Check:  resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "id"),
			},
			{
				Config: nvmetHostSubsysConfigAlt(hostnqn, hostnqnAlt, subsysName),
				Check:  resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "id"),
			},
		},
	})
}

func TestAccNVMetHostSubsysResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	hostnqn := fmt.Sprintf("nqn.2014-08.org.nvmexpress:uuid:%s", randomUUID(t))
	subsysName := fmt.Sprintf("acchsd%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetHostSubsysConfig(hostnqn, subsysName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_nvmet_host_subsys.test"]
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
						return c.DeleteNVMetHostSubsys(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
