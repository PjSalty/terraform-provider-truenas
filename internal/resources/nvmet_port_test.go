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

func TestAccNVMetPort_basic(t *testing.T) {
	resourceName := "truenas_nvmet_port.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetPortDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetPortConfigBasic("127.0.0.1", 4420),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "addr_trtype", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "addr_traddr", "127.0.0.1"),
					resource.TestCheckResourceAttr(resourceName, "addr_trsvcid", "4420"),
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

func testAccCheckNVMetPortDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_port ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("nvmet_port ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetNVMetPort(ctx, id)
		if err == nil {
			return fmt.Errorf("nvmet_port %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of nvmet_port %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetPortExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetNVMetPort(ctx, id); err != nil {
			return fmt.Errorf("nvmet_port %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetPortDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteNVMetPort(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of nvmet_port %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccNVMetPort_disappears(t *testing.T) {
	resourceName := "truenas_nvmet_port.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetPortDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetPortConfigBasic("0.0.0.0", 4420),
				Check:  testAccCheckNVMetPortExists(resourceName),
			},
			{
				Config:             testAccNVMetPortConfigBasic("0.0.0.0", 4420),
				Check:              testAccCheckNVMetPortDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccNVMetPortConfigBasic(addr string, port int) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = %q
  addr_trsvcid = %d
}
`, addr, port)
}
