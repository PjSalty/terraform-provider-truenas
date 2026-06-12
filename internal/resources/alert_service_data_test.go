package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAlertServiceDataSource_basic creates an alert service via
// the resource path then reads it via the datasource. Validates
// settings_json round-trips with the same JSON shape.
func TestAccAlertServiceDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_alert_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertServiceDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-acc-alertsvc-ds"),
					resource.TestCheckResourceAttr(dataSourceName, "type", "Mail"),
					resource.TestCheckResourceAttr(dataSourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "level", "INFO"),
				),
			},
		},
	})
}

func testAccAlertServiceDataSourceConfig() string {
	return `
provider "truenas" {}

resource "truenas_alert_service" "fixture" {
  name    = "tf-acc-alertsvc-ds"
  type    = "Mail"
  enabled = true
  level   = "INFO"

  settings_json = jsonencode({
    email = "alerts@example.com"
  })
}

data "truenas_alert_service" "test" {
  id = truenas_alert_service.fixture.id
}
`
}

func TestAccAlertServiceDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_alert_service" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
