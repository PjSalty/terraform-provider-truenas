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

func TestAccNVMetHost_basic(t *testing.T) {
	resourceName := "truenas_nvmet_host.test"
	hostnqn := "nqn.2014-08.org.nvmexpress:uuid:acctest-host-0001"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckNVMetAvailable(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetHostDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetHostConfigBasic(hostnqn),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "hostnqn", hostnqn),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// dhchap_* secrets are write-only from the API side; skip
				// those attributes in import state verify if needed.
				ImportStateVerifyIgnore: []string{"dhchap_key", "dhchap_ctrl_key"},
			},
		},
	})
}

func testAccCheckNVMetHostDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("nvmet_host ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("nvmet_host ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetNVMetHost(ctx, id)
		if err == nil {
			return fmt.Errorf("nvmet_host %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of nvmet_host %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetHostExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetNVMetHost(ctx, id); err != nil {
			return fmt.Errorf("nvmet_host %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNVMetHostDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteNVMetHost(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of nvmet_host %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccNVMetHost_disappears(t *testing.T) {
	hostnqn := fmt.Sprintf("nqn.2014-08.com.acctest:disappears-%d", acctest.ShortSuffix())
	resourceName := "truenas_nvmet_host.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNVMetHostDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNVMetHostConfigBasic(hostnqn),
				Check:  testAccCheckNVMetHostExists(resourceName),
			},
			{
				Config:             testAccNVMetHostConfigBasic(hostnqn),
				Check:              testAccCheckNVMetHostDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccNVMetHostConfigBasic(hostnqn string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_nvmet_host" "test" {
  hostnqn = %q
}
`, hostnqn)
}
