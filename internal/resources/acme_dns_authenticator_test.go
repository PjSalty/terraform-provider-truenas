package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
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
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The `attributes` map round-trips through the API as
				// stored, so the import path verifies the same JSON
				// shape the user posted on create.
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
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ACME DNS authenticator ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetACMEDNSAuthenticator(ctx, id)
		if err == nil {
			return fmt.Errorf("ACME DNS authenticator %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of ACME DNS authenticator %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckACMEDNSAuthenticatorExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetACMEDNSAuthenticator(ctx, id); err != nil {
			return fmt.Errorf("ACME DNS authenticator %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckACMEDNSAuthenticatorDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteACMEDNSAuthenticator(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of ACME DNS authenticator %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccACMEDNSAuthenticator_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-acme-disappears-%d", acctest.ShortSuffix())
	resourceName := "truenas_acme_dns_authenticator.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACMEDNSAuthenticatorDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccACMEDNSAuthenticatorConfigBasic(name, "cloudflare", "tf-acc-disappears-token"),
				Check:  testAccCheckACMEDNSAuthenticatorExists(resourceName),
			},
			{
				Config:             testAccACMEDNSAuthenticatorConfigBasic(name, "cloudflare", "tf-acc-disappears-token"),
				Check:              testAccCheckACMEDNSAuthenticatorDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
