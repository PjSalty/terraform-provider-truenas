package resources_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccReportingExporter_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_REPORTING_EXPORTER") != "1" {
		t.Skip("TRUENAS_TEST_REPORTING_EXPORTER=1 not set; skipping")
	}

	resourceName := "truenas_reporting_exporter.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReportingExporterDestroy(resourceName),
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

func testAccCheckReportingExporterExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetReportingExporter(ctx, id); err != nil {
			return fmt.Errorf("reporting exporter %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckReportingExporterDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteReportingExporter(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of reporting exporter %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccReportingExporter_disappears(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_REPORTING_EXPORTER") != "1" {
		t.Skip("TRUENAS_TEST_REPORTING_EXPORTER=1 not set; skipping")
	}
	resourceName := "truenas_reporting_exporter.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReportingExporterDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccReportingExporterConfigBasic("tf-acc-graphite-disappears", false),
				Check:  testAccCheckReportingExporterExists(resourceName),
			},
			{
				Config:             testAccReportingExporterConfigBasic("tf-acc-graphite-disappears", false),
				Check:              testAccCheckReportingExporterDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testAccCheckReportingExporterDestroy verifies the reporting exporter
// is gone from the upstream after Terraform removes it.
func testAccCheckReportingExporterDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("reporting_exporter ID not set on %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("reporting_exporter ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetReportingExporter(ctx, id)
		if err == nil {
			return fmt.Errorf("reporting_exporter %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of reporting_exporter %d: %w", id, err)
		}
		return nil
	}
}
