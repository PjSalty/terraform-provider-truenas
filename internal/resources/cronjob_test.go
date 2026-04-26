package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		return nil
	}
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
