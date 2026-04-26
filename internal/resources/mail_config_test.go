package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMailConfig_basic(t *testing.T) {
	resourceName := "truenas_mail_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with minimal required settings
			{
				Config: testAccMailConfigMinimal(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "1"),
					resource.TestCheckResourceAttr(resourceName, "port", "25"),
					resource.TestCheckResourceAttr(resourceName, "security", "PLAIN"),
					resource.TestCheckResourceAttr(resourceName, "smtp", "false"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "1",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccMailConfig_update(t *testing.T) {
	resourceName := "truenas_mail_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with custom settings
			{
				Config: testAccMailConfigCustom("test@example.com", "smtp.example.com", 587),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "fromemail", "test@example.com"),
					resource.TestCheckResourceAttr(resourceName, "outgoingserver", "smtp.example.com"),
					resource.TestCheckResourceAttr(resourceName, "port", "587"),
				),
			},
			// Update settings
			{
				Config: testAccMailConfigCustom("admin@example.com", "mail.example.com", 25),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "fromemail", "admin@example.com"),
					resource.TestCheckResourceAttr(resourceName, "outgoingserver", "mail.example.com"),
					resource.TestCheckResourceAttr(resourceName, "port", "25"),
				),
			},
		},
	})
}

func testAccMailConfigMinimal() string {
	return `
provider "truenas" {}

resource "truenas_mail_config" "test" {
  fromemail = "tf-test@example.com"
}
`
}

func testAccMailConfigCustom(fromemail, outgoingserver string, port int) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_mail_config" "test" {
  fromemail      = %q
  outgoingserver = %q
  port           = %d
}
`, fromemail, outgoingserver, port)
}
