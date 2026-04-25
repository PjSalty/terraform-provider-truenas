package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKMIPConfig_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_KMIP") != "1" {
		t.Skip("TRUENAS_TEST_KMIP=1 not set; skipping (KMIP is a singleton that affects key management)")
	}

	resourceName := "truenas_kmip_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKMIPConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "kmip"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateId:           "kmip",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"change_server", "validate", "force_clear"},
			},
		},
	})
}

func testAccKMIPConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_kmip_config" "test" {
  enabled          = false
  manage_sed_disks = false
  manage_zfs_keys  = false
  port             = 5696
  server           = ""
  ssl_version      = "PROTOCOL_TLSv1_2"
  validate         = false
}
`
}
