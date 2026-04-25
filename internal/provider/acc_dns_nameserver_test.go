package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDNSNameserverResource_basic — singleton: DNS nameserver
// config is part of network_config. Delete is a no-op reset (TrueNAS
// keeps whatever value is currently set), so import + destroy are not
// meaningful for this singleton.
func TestAccDNSNameserverResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dns_nameserver" "test" {
  nameserver1 = "1.1.1.1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dns_nameserver.test", "nameserver1", "1.1.1.1"),
					resource.TestCheckResourceAttrSet("truenas_dns_nameserver.test", "id"),
				),
			},
		},
	})
}

// TestAccDNSNameserverResource_update toggles nameserver1 between two
// well-known public resolvers, then restores it to the default of
// 1.1.1.1 so the test VM is left in a predictable state. A third step
// ensures the test is idempotent and reverts any mutation it made.
//
// No _disappears test exists because this is a singleton: the backend
// has no API to "delete" it — Delete is a no-op reset — so there is
// nothing meaningful to test for out-of-band deletion drift.
func TestAccDNSNameserverResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dns_nameserver" "test" {
  nameserver1 = "1.1.1.1"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_dns_nameserver.test", "nameserver1", "1.1.1.1"),
			},
			{
				Config: `
resource "truenas_dns_nameserver" "test" {
  nameserver1 = "8.8.8.8"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_dns_nameserver.test", "nameserver1", "8.8.8.8"),
			},
			{
				// Restore the default so the shared test VM is left in a
				// predictable state for the next test run.
				Config: `
resource "truenas_dns_nameserver" "test" {
  nameserver1 = "1.1.1.1"
}
`,
				Check: resource.TestCheckResourceAttr("truenas_dns_nameserver.test", "nameserver1", "1.1.1.1"),
			},
		},
	})
}
