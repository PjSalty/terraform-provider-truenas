package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccAlertServiceResource_basic uses a Mail alert service with dummy
// settings. TrueNAS accepts Mail config without attempting to send a
// test message at create time. settings_json is sensitive and excluded
// from ImportStateVerify because the API masks the password on read.
func TestAccAlertServiceResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-alerts-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_alert_service" "test" {
  name    = %q
  type    = "Mail"
  level   = "WARNING"
  enabled = false
  settings_json = jsonencode({
    email = "root@localhost"
  })
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_alert_service.test", "name", name),
					resource.TestCheckResourceAttr("truenas_alert_service.test", "type", "Mail"),
					resource.TestCheckResourceAttrSet("truenas_alert_service.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_alert_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"settings_json"},
			},
		},
	})
}

// TestAccAlertServiceResource_update changes the alert level in place
// without touching `type`, which would otherwise require replacement.
func TestAccAlertServiceResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-alerts-u-%d", shortSuffix())
	cfg := func(level string) string {
		return fmt.Sprintf(`
resource "truenas_alert_service" "test" {
  name    = %q
  type    = "Mail"
  level   = %q
  enabled = false
  settings_json = jsonencode({
    email = "root@localhost"
  })
}
`, name, level)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("WARNING"),
				Check:  resource.TestCheckResourceAttr("truenas_alert_service.test", "level", "WARNING"),
			},
			{
				Config: cfg("CRITICAL"),
				Check:  resource.TestCheckResourceAttr("truenas_alert_service.test", "level", "CRITICAL"),
			},
		},
	})
}

// TestAccAlertServiceResource_disappears deletes the alert service
// out-of-band to verify the provider detects drift on refresh.
func TestAccAlertServiceResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-alerts-d-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_alert_service" "test" {
  name    = %q
  type    = "Mail"
  level   = "WARNING"
  enabled = false
  settings_json = jsonencode({
    email = "root@localhost"
  })
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_alert_service.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_alert_service.test"]
						if !ok {
							return fmt.Errorf("truenas_alert_service.test not in state")
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
						return c.DeleteAlertService(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
