package resources_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestAccReportingExporter_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_REPORTING_EXPORTER") != "1" {
		t.Skip("TRUENAS_TEST_REPORTING_EXPORTER=1 not set; skipping")
	}

	resourceName := "truenas_reporting_exporter.test"
	// Unique per run. A fixed name meant one failed run left an exporter
	// behind and every later run died on
	// "reporting_exporter_create.name: Specified name is already in use",
	// which reads like a provider fault rather than leftover state.
	name := acctest.RandomName("tfaccgraphite")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReportingExporterDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccReportingExporterConfigBasic(name, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The server fills in its own defaults for an exporter, so a
				// round-tripped attributes_json carries buffer_on_failures,
				// matching_charts, send_names_instead_of_ids and update_every
				// that the configuration never set. Comparing a sparse config
				// against an enriched read can only fail. The same attribute is
				// ignored for the same reason in
				// internal/provider/acc_reporting_exporter_test.go.
				ImportStateVerifyIgnore: []string{"attributes_json"},
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
	disappearsName := acctest.RandomName("tfaccgraphitedis")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReportingExporterDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccReportingExporterConfigBasic(disappearsName, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: testAccCheckReportingExporterExists(resourceName),
			},
			{
				Config:             testAccReportingExporterConfigBasic(disappearsName, false),
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
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of reporting_exporter %d: %w", id, err)
		}
		return nil
	}
}
