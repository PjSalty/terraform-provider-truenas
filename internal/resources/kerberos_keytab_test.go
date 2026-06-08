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

// validKeytabB64 is a minimal MIT keytab v2 fixture that TrueNAS
// SCALE 25.10's keytab parser accepts. SCALE 25.04 accepted any
// base64 blob; 25.10+ validates the structure and rejects garbage
// with "File does not contain any keytab entries".
//
// Wire layout (constructed in scripts/gen_test_keytab.py, kept in
// sync if you regenerate):
//
//	magic       = 0x05 0x02                (MIT keytab v2)
//	entry_size  = int32 BE                  (signed; len of payload)
//	payload:
//	    num_components       = 1            (excludes realm)
//	    realm                = "EXAMPLE.COM"
//	    component            = "testuser"
//	    name_type            = 1            (NT-PRINCIPAL)
//	    timestamp            = 0
//	    vno8                 = 1
//	    enctype              = 18           (AES256-CTS-HMAC-SHA1-96)
//	    key                  = 32 bytes of zeros
//
// The key is all zeros — fine for a fixture; the test path never
// actually authenticates with this keytab.
const validKeytabB64 = "BQIAAABGAAEAC0VYQU1QTEUuQ09NAAh0ZXN0dXNlcgAAAAEAAAAAAQASACAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="

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
