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

// validKeytabB64 is a minimal MIT-format keytab fixture that parses as
// real keytab data. SCALE 25.04 accepted any base64 blob; 25.10+ validates
// the parse and rejects garbage. Contents are dummy (principal
// testuser@EXAMPLE.COM.INVALID, zero key); fine for fixture use, the
// keytab is never actually exercised for auth in the test path.
const validKeytabB64 = "BQIAAABFAAEAEUVYQU1QTEUuQ09NLklOVkFMSUQAB3Rlc3R1c2VyAAAAAVwOJWQBABIAIN5w" +
	"M+JFu6n9P3JfDe5N9bDM3JIuVxoVNvBYdVu8/JqLAAAAAg=="

func TestAccKerberosKeytab_basic(t *testing.T) {
	resourceName := "truenas_kerberos_keytab.test"
	payload := validKeytabB64

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
	priv := validKeytabB64

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
