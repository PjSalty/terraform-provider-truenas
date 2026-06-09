package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestAccNVMetPortSubsys_basic(t *testing.T) {
	resourceName := "truenas_nvmet_port_subsys.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetPortSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetPortSubsysConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "port_id"),
					resource.TestCheckResourceAttrSet(resourceName, "subsys_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckNVMetPortSubsysDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_port_subsys ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("nvmet_port_subsys ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetNVMetPortSubsys(ctx, id)
		if err == nil {
			return fmt.Errorf("nvmet_port_subsys %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of nvmet_port_subsys %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetPortSubsysExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetNVMetPortSubsys(ctx, id); err != nil {
			return fmt.Errorf("nvmet_port_subsys %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetPortSubsysDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteNVMetPortSubsys(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of nvmet_port_subsys %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccNVMetPortSubsys_disappears(t *testing.T) {
	resourceName := "truenas_nvmet_port_subsys.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetPortSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetPortSubsysConfigBasic(),
				Check:  testAccCheckNVMetPortSubsysExists(resourceName),
			},
			{
				Config:             testAccNVMetPortSubsysConfigBasic(),
				Check:              testAccCheckNVMetPortSubsysDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccNVMetPortSubsysConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_nvmet_port" "ps_port" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = 4420
}

resource "truenas_nvmet_subsys" "ps_subsys" {
  name = "tf-acc-ps-subsys"
}

resource "truenas_nvmet_port_subsys" "test" {
  port_id   = tonumber(truenas_nvmet_port.ps_port.id)
  subsys_id = tonumber(truenas_nvmet_subsys.ps_subsys.id)
}
`
}
