package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories is used for acceptance testing — it
// wires up a real in-process provider server without a network round-trip.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"truenas": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies the environment is configured for acceptance
// testing against a real TrueNAS SCALE instance. Tests are skipped unless
// TF_ACC=1 AND both TRUENAS_URL and TRUENAS_API_KEY are set.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TRUENAS_URL") == "" {
		t.Fatal("TRUENAS_URL must be set for acceptance tests")
	}
	if os.Getenv("TRUENAS_API_KEY") == "" {
		t.Fatal("TRUENAS_API_KEY must be set for acceptance tests")
	}
}

// TestAccProvider_Schema is a minimal smoke test that exercises the
// provider factory and schema validation under the TF acceptance harness.
// Run with: TF_ACC=1 TRUENAS_URL=... TRUENAS_API_KEY=... go test ./internal/provider/...
func TestAccProvider_Schema(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set — skipping acceptance test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_system_info" "this" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_system_info.this", "version"),
					resource.TestCheckResourceAttrSet("data.truenas_system_info.this", "hostname"),
				),
			},
		},
	})
}
