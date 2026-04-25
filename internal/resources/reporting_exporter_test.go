package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReportingExporter_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_REPORTING_EXPORTER") != "1" {
		t.Skip("TRUENAS_TEST_REPORTING_EXPORTER=1 not set; skipping")
	}

	resourceName := "truenas_reporting_exporter.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReportingExporterConfigBasic("tf-acc-graphite", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-graphite"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
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

func testAccReportingExporterConfigBasic(name string, enabled bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_reporting_exporter" "test" {
  name    = %q
  enabled = %t
  attributes_json = jsonencode({
    exporter_type    = "GRAPHITE"
    destination_ip   = "127.0.0.1"
    destination_port = 2003
    namespace        = "truenas"
    prefix           = "scale"
  })
}
`, name, enabled)
}
