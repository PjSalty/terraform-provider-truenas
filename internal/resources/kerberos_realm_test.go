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
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("kerberos realm ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetKerberosRealm(ctx, id)
		if err == nil {
			return fmt.Errorf("kerberos realm %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of kerberos realm %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckKerberosRealmExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetKerberosRealm(ctx, id); err != nil {
			return fmt.Errorf("kerberos realm %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckKerberosRealmDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteKerberosRealm(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of kerberos realm %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccKerberosRealm_disappears(t *testing.T) {
	resourceName := "truenas_kerberos_realm.test"
	realm := fmt.Sprintf("TF-ACC-DISAPPEARS-%d.EXAMPLE.COM", acctest.ShortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKerberosRealmDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccKerberosRealmConfigBasic(realm),
				Check:  testAccCheckKerberosRealmExists(resourceName),
			},
			{
				Config:             testAccKerberosRealmConfigBasic(realm),
				Check:              testAccCheckKerberosRealmDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
