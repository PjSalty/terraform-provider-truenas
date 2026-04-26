package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCloudSyncCredential_schemaValidation exercises the full
// Plugin Framework Plan phase end-to-end for truenas_cloudsync_credential:
// schema compilation, HCL → config parsing, validator pass, plan
// computation. It runs under PlanOnly + ExpectNonEmptyPlan so the
// test passes in any environment where the provider can be loaded —
// no live TrueNAS, no cloud provider credentials needed.
//
// This is the same shallow-acceptance-test shape used by
// TestAccCloudSync_schemaValidation and TestAccCloudBackup_schemaValidation.
// The Phase J coverage ratchet (TestAcceptanceTestCoverage in internal/provider/)
// enforces that EVERY resource carries at least one TestAcc function;
// this file closes the last gap (cloudsync_credential was the sole
// resource without any TestAcc as of v1.7.0).
func TestAccCloudSyncCredential_schemaValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping acceptance test in short mode")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testAccCloudSyncCredentialConfigBasic(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCloudSyncCredentialConfigBasic() string {
	// S3-shape credential is the canonical form; other provider types
	// (B2, Azure, GCS) would substitute their own provider_attributes_json
	// shape. The value is bogus on purpose — PlanOnly never writes.
	return `
provider "truenas" {}

resource "truenas_cloudsync_credential" "test" {
  name          = "tf-acc-test-s3"
  provider_type = "S3"
  provider_attributes_json = jsonencode({
    access_key_id     = "AKIAIOSFODNN7EXAMPLE"
    secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  })
}
`
}
