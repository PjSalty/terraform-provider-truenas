package resources_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccKerberosKeytab_basic(t *testing.T) {
	resourceName := "truenas_kerberos_keytab.test"
	// A dummy base64 payload — real keytabs are opaque binary.
	payload := base64.StdEncoding.EncodeToString([]byte("dummy-tf-acc-keytab"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKerberosKeytabDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccKerberosKeytabConfig("tf-acc-keytab", payload),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-keytab"),
				),
			},
			// Import — file is sensitive and may round-trip with re-encoding,
			// so exclude it from the import diff check.
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"file"},
			},
		},
	})
}

func testAccCheckKerberosKeytabDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("kerberos keytab ID not set")
		}
		return nil
	}
}

func testAccKerberosKeytabConfig(name, b64 string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_kerberos_keytab" "test" {
  name = %q
  file = %q
}
`, name, b64)
}
