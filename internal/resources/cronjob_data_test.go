package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCronJobDataSource_basic creates a fixture cron job via the
// resource path then looks it up via the datasource. Asserts the
// returned attributes round-trip cleanly. Catches a regression where
// the datasource Read path stops mapping a field that the resource
// path still writes.
func TestAccCronJobDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_cronjob.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCronJobDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "user", "root"),
					resource.TestCheckResourceAttr(dataSourceName, "command", "/bin/true"),
					resource.TestCheckResourceAttr(dataSourceName, "description", "tf-acc cronjob ds"),
					resource.TestCheckResourceAttr(dataSourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "schedule_minute", "*/15"),
					resource.TestCheckResourceAttr(dataSourceName, "schedule_hour", "*"),
				),
			},
		},
	})
}

func testAccCronJobDataSourceConfig() string {
	return `
provider "truenas" {}

resource "truenas_cronjob" "fixture" {
  user            = "root"
  command         = "/bin/true"
  description     = "tf-acc cronjob ds"
  enabled         = true
  schedule_minute = "*/15"
  schedule_hour   = "*"
  schedule_dom    = "*"
  schedule_month  = "*"
  schedule_dow    = "*"
}

data "truenas_cronjob" "test" {
  id = truenas_cronjob.fixture.id
}
`
}

// TestAccCronJobDataSource_notFound asserts the datasource Read path
// surfaces a clean "not found" diagnostic when the operator asks for
// an ID that doesn't exist on the server. Exercises the same path
// that would break on a v2.0 datasource if its Read implementation
// regressed to swallowing 404s.
func TestAccCronJobDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCronJobDataSourceConfigNotFound(99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}

func testAccCronJobDataSourceConfigNotFound(id int) string {
	return fmt.Sprintf(`
provider "truenas" {}

data "truenas_cronjob" "test" {
  id = %d
}
`, id)
}
