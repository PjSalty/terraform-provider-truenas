package resources_test

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
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
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("kerberos keytab ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetKerberosKeytab(ctx, id)
		if err == nil {
			return fmt.Errorf("kerberos keytab %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of kerberos keytab %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckKerberosKeytabExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetKerberosKeytab(ctx, id); err != nil {
			return fmt.Errorf("kerberos keytab %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckKerberosKeytabDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteKerberosKeytab(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of kerberos keytab %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccKerberosKeytab_disappears(t *testing.T) {
	resourceName := "truenas_kerberos_keytab.test"
	name := fmt.Sprintf("tf-acc-keytab-disappears-%d", acctest.ShortSuffix())
	// Minimal MIT-format keytab fixture (same shape the basic test uses).
	const keytabB64 = "BQIAAABFAAEAEUVYQU1QTEUuQ09NLklOVkFMSUQAB3Rlc3R1c2VyAAAAAVwOJWQBABIAIN5w" +
		"M+JFu6n9P3JfDe5N9bDM3JIuVxoVNvBYdVu8/JqLAAAAAg=="
	_ = keytabB64
	priv := base64.StdEncoding.EncodeToString([]byte("disappears-keytab-fixture"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKerberosKeytabDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccKerberosKeytabConfig(name, priv),
				Check:  testAccCheckKerberosKeytabExists(resourceName),
			},
			{
				Config:             testAccKerberosKeytabConfig(name, priv),
				Check:              testAccCheckKerberosKeytabDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
