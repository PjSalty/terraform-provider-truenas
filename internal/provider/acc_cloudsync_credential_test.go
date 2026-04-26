package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccCloudSyncCredentialResource_basic — TrueNAS does NOT probe S3
// credentials at create time for /cloudsync/credentials (the probe is
// deferred to cloudsync task create). provider_attributes_json is
// sensitive and excluded from ImportStateVerify.
func TestAccCloudSyncCredentialResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-csc-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_cloudsync_credential" "test" {
  name          = %q
  provider_type = "S3"
  provider_attributes_json = jsonencode({
    access_key_id     = "AKIAACCTESTFAKEKEY01"
    secret_access_key = "acctestFakeSecretAccessKeyDoNotUsePlease"
  })
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cloudsync_credential.test", "name", name),
					resource.TestCheckResourceAttr("truenas_cloudsync_credential.test", "provider_type", "S3"),
					resource.TestCheckResourceAttrSet("truenas_cloudsync_credential.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_cloudsync_credential.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"provider_attributes_json"},
			},
		},
	})
}

// TestAccCloudSyncCredentialResource_update renames an existing S3
// credential in place. Two steps are sufficient — the resource is a
// regular (not singleton) resource and the test cleans itself up on
// exit.
func TestAccCloudSyncCredentialResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	base := fmt.Sprintf("acctest-csc-upd-%d", shortSuffix())
	cfg := func(name string) string {
		return fmt.Sprintf(`
resource "truenas_cloudsync_credential" "test" {
  name          = %q
  provider_type = "S3"
  provider_attributes_json = jsonencode({
    access_key_id     = "AKIAACCTESTFAKEKEY02"
    secret_access_key = "acctestFakeSecretAccessKeyDoNotUsePlease"
  })
}
`, name)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(base + "-a"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_cloudsync_credential.test", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_cloudsync_credential.test", "name", base+"-a"),
			},
			{
				Config: cfg(base + "-b"),
				// Renaming `name` MUST be an in-place Update. This
				// guards against any future schema change that would
				// accidentally mark `name` as RequiresReplace.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_cloudsync_credential.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_cloudsync_credential.test", "name", base+"-b"),
			},
		},
	})
}

// TestAccCloudSyncCredentialResource_disappears deletes the credential
// out-of-band via the TrueNAS API and verifies the provider detects
// the drift and produces a non-empty plan on the next refresh.
func TestAccCloudSyncCredentialResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctest-csc-dis-%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_cloudsync_credential" "test" {
  name          = %q
  provider_type = "S3"
  provider_attributes_json = jsonencode({
    access_key_id     = "AKIAACCTESTFAKEKEY03"
    secret_access_key = "acctestFakeSecretAccessKeyDoNotUsePlease"
  })
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_cloudsync_credential.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_cloudsync_credential.test"]
						if !ok {
							return fmt.Errorf("truenas_cloudsync_credential.test not found in state")
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
						return c.DeleteCloudSyncCredential(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
