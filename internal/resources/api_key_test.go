package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccAPIKey_basic(t *testing.T) {
	resourceName := "truenas_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyConfigBasic("tf-acc-test-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-key"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "username"),
				),
			},
		},
	})
}

func TestAccAPIKey_update(t *testing.T) {
	resourceName := "truenas_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyConfigBasic("tf-acc-test-key-v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-key-v1"),
				),
			},
			{
				Config: testAccAPIKeyConfigBasic("tf-acc-test-key-v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-key-v2"),
				),
			},
		},
	})
}

func testAccCheckAPIKeyDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("API key ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("API key ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetAPIKey(ctx, id)
		if err == nil {
			return fmt.Errorf("API key %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of API key %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckAPIKeyExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetAPIKey(ctx, id); err != nil {
			return fmt.Errorf("API key %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckAPIKeyDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteAPIKey(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of API key %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccAPIKey_disappears(t *testing.T) {
	resourceName := "truenas_api_key.test"
	name := fmt.Sprintf("tf-acc-apikey-disappears-%d", acctest.ShortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAPIKeyDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyConfigBasic(name),
				Check:  testAccCheckAPIKeyExists(resourceName),
			},
			{
				Config:             testAccAPIKeyConfigBasic(name),
				Check:              testAccCheckAPIKeyDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAPIKeyConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_api_key" "test" {
  name     = %q
  username = "truenas_admin"
}
`, name)
}
