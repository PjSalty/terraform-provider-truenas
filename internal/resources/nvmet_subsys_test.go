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

func TestAccNVMetSubsys_basic(t *testing.T) {
	resourceName := "truenas_nvmet_subsys.test"
	name := "tf-acc-nvmet-subsys"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetSubsysConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "subnqn"),
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

func testAccCheckNVMetSubsysDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_subsys ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("nvmet_subsys ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetNVMetSubsys(ctx, id)
		if err == nil {
			return fmt.Errorf("nvmet_subsys %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of nvmet_subsys %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetSubsysExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetNVMetSubsys(ctx, id); err != nil {
			return fmt.Errorf("nvmet_subsys %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetSubsysDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteNVMetSubsys(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of nvmet_subsys %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccNVMetSubsys_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-nvmet-subsys-disappears-%d", acctest.ShortSuffix())
	resourceName := "truenas_nvmet_subsys.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetSubsysDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetSubsysConfigBasic(name),
				Check:  testAccCheckNVMetSubsysExists(resourceName),
			},
			{
				Config:             testAccNVMetSubsysConfigBasic(name),
				Check:              testAccCheckNVMetSubsysDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccNVMetSubsysConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nvmet_subsys" "test" {
  name = %q
}
`, name)
}
