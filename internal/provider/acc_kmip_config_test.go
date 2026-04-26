package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccKMIPConfigResource_basic — singleton: KMIP service configuration
// has a single instance and Delete is a no-op reset. enabled is left
// false and server is empty so the test VM never tries to reach a real
// KMIP server and no encryption keys are migrated.
func TestAccKMIPConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_kmip_config" "test" {
  enabled          = false
  manage_sed_disks = false
  manage_zfs_keys  = false
  port             = 5696
  server           = ""
  ssl_version      = "PROTOCOL_TLSv1_2"
  validate         = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_kmip_config.test", "enabled", "false"),
					resource.TestCheckResourceAttr("truenas_kmip_config.test", "port", "5696"),
					resource.TestCheckResourceAttrSet("truenas_kmip_config.test", "id"),
				),
			},
		},
	})
}

// TestAccKMIPConfigResource_update is intentionally a no-op skip.
//
// SCALE 25.10's KMIP service does not persist ANY user-set field while
// `enabled=false` — the API normalizes the request body but the underlying
// store keeps the defaults until KMIP is actually enabled and a real KMIP
// server is reachable. Both `port` and `ssl_version` were observed to
// silently drift back to the defaults across every Update attempt.
// Enabling KMIP would require provisioning a real KMIP server which is
// outside the scope of acceptance tests run against an isolated test VM,
// so the update path is verified at the unit-test level instead (see
// internal/resources/kmip_config_test.go).
func TestAccKMIPConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	t.Skip("KMIP update path requires a real KMIP server (enabled=true); covered by unit tests instead")
}
