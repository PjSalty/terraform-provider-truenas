package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccReportingExporterResource_basic uses a Graphite exporter
// pointed at a dummy address. TrueNAS does not test-connect to the
// exporter at create time.
func TestAccReportingExporterResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-exp-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_reporting_exporter" "test" {
  name    = %q
  enabled = false
  attributes_json = jsonencode({
    exporter_type    = "GRAPHITE"
    destination_ip   = "127.0.0.1"
    destination_port = 2003
    namespace        = "truenas"
    prefix           = "servers"
  })
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_reporting_exporter.test", "name", name),
					resource.TestCheckResourceAttrSet("truenas_reporting_exporter.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_reporting_exporter.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"attributes_json"},
			},
		},
	})
}

// TestAccReportingExporterResource_update flips the enabled flag on an
// existing Graphite exporter. Two steps are sufficient — the resource
// is regular (not a singleton) and the test cleans itself up on exit.
func TestAccReportingExporterResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-exp-upd-%d", shortSuffix())
	cfg := func(enabled bool) string {
		return fmt.Sprintf(`
resource "truenas_reporting_exporter" "test" {
  name    = %q
  enabled = %t
  attributes_json = jsonencode({
    exporter_type    = "GRAPHITE"
    destination_ip   = "127.0.0.1"
    destination_port = 2003
    namespace        = "truenas"
    prefix           = "servers"
  })
}
`, name, enabled)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_reporting_exporter.test", "name", name),
					resource.TestCheckResourceAttr("truenas_reporting_exporter.test", "enabled", "false"),
				),
			},
			{
				Config: cfg(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_reporting_exporter.test", "name", name),
					resource.TestCheckResourceAttr("truenas_reporting_exporter.test", "enabled", "true"),
				),
			},
		},
	})
}

// TestAccReportingExporterResource_disappears deletes the exporter
// out-of-band via the TrueNAS API and verifies that the provider
// detects the drift and produces a non-empty plan on the next refresh.
func TestAccReportingExporterResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-exp-dis-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_reporting_exporter" "test" {
  name    = %q
  enabled = false
  attributes_json = jsonencode({
    exporter_type    = "GRAPHITE"
    destination_ip   = "127.0.0.1"
    destination_port = 2003
    namespace        = "truenas"
    prefix           = "servers"
  })
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_reporting_exporter.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_reporting_exporter.test"]
						if !ok {
							return fmt.Errorf("truenas_reporting_exporter.test not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parse id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteReportingExporter(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
