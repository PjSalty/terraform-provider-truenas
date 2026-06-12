package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAPIKeyDataSource_basic creates an API key via the resource
// path then reads it back via the datasource. The datasource never
// exposes the secret material (api_key.key is write-only on the
// resource side and absent from the datasource side); this test
// validates the non-secret attributes round-trip cleanly.
func TestAccAPIKeyDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-acc-apikey-ds"),
					resource.TestCheckResourceAttrSet(dataSourceName, "username"),
					// The actual API key string is sensitive and only
					// returned at creation time. Datasource must not
					// expose it.
					resource.TestCheckNoResourceAttr(dataSourceName, "key"),
				),
			},
		},
	})
}

func testAccAPIKeyDataSourceConfig() string {
	return `
provider "truenas" {}

resource "truenas_api_key" "fixture" {
  name     = "tf-acc-apikey-ds"
  username = "truenas_admin"
}

data "truenas_api_key" "test" {
  id = truenas_api_key.fixture.id
}
`
}

// TestAccAPIKeyDataSource_notFound, sanity check that a missing
// API key id surfaces as an error, not a silent empty result.
func TestAccAPIKeyDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_api_key" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
