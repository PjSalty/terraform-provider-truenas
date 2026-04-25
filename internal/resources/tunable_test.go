package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccTunable_basic(t *testing.T) {
	resourceName := "truenas_tunable.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTunableDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccTunableConfigBasic("SYSCTL", "net.ipv4.ip_forward", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "SYSCTL"),
					resource.TestCheckResourceAttr(resourceName, "var", "net.ipv4.ip_forward"),
					resource.TestCheckResourceAttr(resourceName, "value", "1"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccTunable_update(t *testing.T) {
	resourceName := "truenas_tunable.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTunableDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccTunableConfigWithComment("SYSCTL", "net.ipv4.tcp_syncookies", "1", "Enable syncookies"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "value", "1"),
					resource.TestCheckResourceAttr(resourceName, "comment", "Enable syncookies"),
				),
			},
			// Update value and comment in-place
			{
				Config: testAccTunableConfigWithComment("SYSCTL", "net.ipv4.tcp_syncookies", "0", "Disable syncookies"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "value", "0"),
					resource.TestCheckResourceAttr(resourceName, "comment", "Disable syncookies"),
				),
			},
		},
	})
}

func testAccCheckTunableDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("tunable ID not set")
		}
		return nil
	}
}

func testAccTunableConfigBasic(tunableType, varName, value string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_tunable" "test" {
  type  = %q
  var   = %q
  value = %q
}
`, tunableType, varName, value)
}

func testAccTunableConfigWithComment(tunableType, varName, value, comment string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_tunable" "test" {
  type    = %q
  var     = %q
  value   = %q
  comment = %q
}
`, tunableType, varName, value, comment)
}
