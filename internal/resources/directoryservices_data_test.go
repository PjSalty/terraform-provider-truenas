package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDirectoryServicesDataSource_basic reads the always-present
// directoryservices singleton. No fixture needed, every TrueNAS
// has a directoryservices config (defaults to enable=false,
// service_type=null on a fresh install).
//
// Behavioural coverage of the join + leave lifecycle lives in
// internal/resources/directoryservices_test.go::
// TestAccDirectoryServices_fullADLifecycle, which is env-gated
// behind TRUENAS_TEST_AD=1. This test is the lightweight
// read-only smoke.
func TestAccDirectoryServicesDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_directoryservices.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_directoryservices" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "directoryservices"),
					resource.TestCheckResourceAttrSet(dataSourceName, "enable"),
				),
			},
		},
	})
}
