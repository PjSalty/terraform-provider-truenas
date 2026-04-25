package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccKerberosRealm_basic(t *testing.T) {
	resourceName := "truenas_kerberos_realm.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKerberosRealmDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccKerberosRealmConfigBasic("TF-ACC.EXAMPLE.COM"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "realm", "TF-ACC.EXAMPLE.COM"),
					resource.TestCheckResourceAttr(resourceName, "kdc.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "kdc.0", "kdc1.tf-acc.example.com"),
				),
			},
			// Update: change KDC list
			{
				Config: testAccKerberosRealmConfigUpdated("TF-ACC.EXAMPLE.COM"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "kdc.#", "2"),
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

func testAccCheckKerberosRealmDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("kerberos realm ID not set")
		}
		return nil
	}
}

func testAccKerberosRealmConfigBasic(realm string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_kerberos_realm" "test" {
  realm = %q
  kdc   = ["kdc1.tf-acc.example.com"]
}
`, realm)
}

func testAccKerberosRealmConfigUpdated(realm string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_kerberos_realm" "test" {
  realm          = %q
  kdc            = ["kdc1.tf-acc.example.com", "kdc2.tf-acc.example.com"]
  admin_server   = ["admin.tf-acc.example.com"]
  kpasswd_server = ["kpasswd.tf-acc.example.com"]
}
`, realm)
}
