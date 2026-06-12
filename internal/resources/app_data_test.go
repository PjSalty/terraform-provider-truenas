package resources_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAppDataSource_basic reads a specific TrueNAS App by id.
// Env-gated because Apps require a dedicated apps pool + a known
// pre-installed app name (TRUENAS_TEST_APP_NAME) — the datasource
// Read path is exercised without the heavyweight install in the
// test.
func TestAccAppDataSource_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_APPS") != "1" {
		t.Skip("set TRUENAS_TEST_APPS=1 and TRUENAS_TEST_APP_NAME to run")
	}
	name := os.Getenv("TRUENAS_TEST_APP_NAME")
	if name == "" {
		t.Skip("set TRUENAS_TEST_APP_NAME to point at an installed app")
	}
	dataSourceName := "data.truenas_app.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_app" "test" {
  id = %q
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "state"),
				),
			},
		},
	})
}

func TestAccAppDataSource_notFound(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_APPS") != "1" {
		t.Skip("set TRUENAS_TEST_APPS=1 to run")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_app" "test" {
  id = "definitely-not-installed-99"
}
`,
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
