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

func TestAccCronJob_basic(t *testing.T) {
	resourceName := "truenas_cronjob.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronJobDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCronJobConfigBasic(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "user", "root"),
					resource.TestCheckResourceAttr(resourceName, "command", "echo tf-acc-test"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "description", "tf-acc-test cronjob"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCronJob_enableDisable(t *testing.T) {
	resourceName := "truenas_cronjob.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronJobDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create enabled
			{
				Config: testAccCronJobConfigBasic(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			// Update: disable
			{
				Config: testAccCronJobConfigBasic(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
		},
	})
}

func testAccCheckCronJobDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("cron job ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("cron job ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetCronJob(ctx, id)
		if err == nil {
			return fmt.Errorf("cron job %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of cron job %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckCronJobExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetCronJob(ctx, id); err != nil {
			return fmt.Errorf("cron job %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckCronJobDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteCronJob(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of cron job %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccCronJob_disappears(t *testing.T) {
	resourceName := "truenas_cronjob.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronJobDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCronJobConfigBasic(true),
				Check:  testAccCheckCronJobExists(resourceName),
			},
			{
				Config:             testAccCronJobConfigBasic(true),
				Check:              testAccCheckCronJobDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCronJobConfigBasic(enabled bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_cronjob" "test" {
  user        = "root"
  command     = "echo tf-acc-test"
  description = "tf-acc-test cronjob"
  enabled     = %t

  schedule_minute = "0"
  schedule_hour   = "0"
}
`, enabled)
}
