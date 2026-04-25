package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccKerberosRealmResource_basic creates a dummy realm. TrueNAS does
// not contact the KDC at create time, so a fake realm under the RFC 2606
// reserved EXAMPLE.COM domain is safe.
func TestAccKerberosRealmResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	realm := fmt.Sprintf("ACCTEST%d.EXAMPLE.COM", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_kerberos_realm" "test" {
  realm = %q
}
`, realm),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_kerberos_realm.test", "realm", realm),
					resource.TestCheckResourceAttrSet("truenas_kerberos_realm.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_kerberos_realm.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccKerberosRealmResource_update toggles primary_kdc while the
// realm stays constant.
func TestAccKerberosRealmResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	realm := fmt.Sprintf("ACCTESTU%d.EXAMPLE.COM", shortSuffix())
	cfg := func(kdc string) string {
		return fmt.Sprintf(`
resource "truenas_kerberos_realm" "test" {
  realm       = %q
  primary_kdc = %q
}
`, realm, kdc)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("kdc1.example.com"),
				Check:  resource.TestCheckResourceAttr("truenas_kerberos_realm.test", "primary_kdc", "kdc1.example.com"),
			},
			{
				Config: cfg("kdc2.example.com"),
				Check:  resource.TestCheckResourceAttr("truenas_kerberos_realm.test", "primary_kdc", "kdc2.example.com"),
			},
		},
	})
}

// TestAccKerberosRealmResource_disappears deletes the realm out-of-band.
func TestAccKerberosRealmResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	realm := fmt.Sprintf("ACCTESTD%d.EXAMPLE.COM", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_kerberos_realm" "test" {
  realm = %q
}
`, realm),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_kerberos_realm.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_kerberos_realm.test"]
						if !ok {
							return fmt.Errorf("truenas_kerberos_realm.test not in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parse id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteKerberosRealm(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
