package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccACMEDNSAuthenticator_basic(t *testing.T) {
	resourceName := "truenas_acme_dns_authenticator.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACMEDNSAuthenticatorDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccACMEDNSAuthenticatorConfigBasic("tf-acc-test-acme", "cloudflare", "fake-token-12345"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-acme"),
					resource.TestCheckResourceAttr(resourceName, "authenticator", "cloudflare"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccACMEDNSAuthenticator_update(t *testing.T) {
	resourceName := "truenas_acme_dns_authenticator.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACMEDNSAuthenticatorDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccACMEDNSAuthenticatorConfigBasic("tf-acc-test-acme-v1", "cloudflare", "fake-token-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-acme-v1"),
				),
			},
			{
				Config: testAccACMEDNSAuthenticatorConfigBasic("tf-acc-test-acme-v2", "cloudflare", "fake-token-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-acme-v2"),
				),
			},
		},
	})
}

func testAccCheckACMEDNSAuthenticatorDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ACME DNS authenticator ID not set")
		}
		return nil
	}
}

func testAccACMEDNSAuthenticatorConfigBasic(name, authenticator, apiToken string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_acme_dns_authenticator" "test" {
  name          = %q
  authenticator = %q
  attributes = {
    api_token = %q
  }
}
`, name, authenticator, apiToken)
}
