package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccNVMetHostSubsys_basic(t *testing.T) {
	resourceName := "truenas_nvmet_host_subsys.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetHostSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetHostSubsysConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "host_id"),
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

func testAccCheckNVMetHostSubsysDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_host_subsys ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("nvmet_host_subsys ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetNVMetHostSubsys(ctx, id)
		if err == nil {
			return fmt.Errorf("nvmet_host_subsys %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of nvmet_host_subsys %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetHostSubsysExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetNVMetHostSubsys(ctx, id); err != nil {
			return fmt.Errorf("nvmet_host_subsys %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetHostSubsysDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteNVMetHostSubsys(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of nvmet_host_subsys %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccNVMetHostSubsys_disappears(t *testing.T) {
	resourceName := "truenas_nvmet_host_subsys.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetHostSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetHostSubsysConfigBasic(),
				Check:  testAccCheckNVMetHostSubsysExists(resourceName),
			},
			{
				Config:             testAccNVMetHostSubsysConfigBasic(),
				Check:              testAccCheckNVMetHostSubsysDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccNVMetHostSubsysConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_nvmet_host" "hs_host" {
  hostnqn = "nqn.2014-08.org.nvmexpress:uuid:acctest-hs-0001"
}

resource "truenas_nvmet_subsys" "hs_subsys" {
  name = "tf-acc-hs-subsys"
}

resource "truenas_nvmet_host_subsys" "test" {
  host_id   = tonumber(truenas_nvmet_host.hs_host.id)
  subsys_id = tonumber(truenas_nvmet_subsys.hs_subsys.id)
}
`
}
