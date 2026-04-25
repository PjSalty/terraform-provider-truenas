package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIAuth_basic(t *testing.T) {
	resourceName := "truenas_iscsi_auth.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIAuthConfigBasic(99, "tf-acc-user", "TfAccSecret12345"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "tag", "99"),
					resource.TestCheckResourceAttr(resourceName, "user", "tf-acc-user"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "peersecret"},
			},
		},
	})
}

func testAccISCSIAuthConfigBasic(tag int, user, secret string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_iscsi_auth" "test" {
  tag    = %d
  user   = %q
  secret = %q
}
`, tag, user, secret)
}
