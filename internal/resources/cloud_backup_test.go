package resources_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccCloudBackup_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_CLOUD_BACKUP") != "1" {
		t.Skip("TRUENAS_TEST_CLOUD_BACKUP=1 not set; skipping (requires cloud credential and restic repo)")
	}

	credID := os.Getenv("TRUENAS_TEST_CLOUD_CREDENTIAL_ID")
	bucket := os.Getenv("TRUENAS_TEST_CLOUD_BUCKET")
	path := os.Getenv("TRUENAS_TEST_CLOUD_PATH")
	password := os.Getenv("TRUENAS_TEST_CLOUD_PASSWORD")
	if credID == "" || bucket == "" || path == "" || password == "" {
		t.Skip("required TRUENAS_TEST_CLOUD_* vars not set")
	}

	resourceName := "truenas_cloud_backup.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCloudBackupDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudBackupConfigBasic(credID, bucket, path, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "path", path),
					resource.TestCheckResourceAttr(resourceName, "keep_last", "5"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccCloudBackupConfigBasic(credID, bucket, path, password string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_cloud_backup" "test" {
  description      = "tf-acc-test cloud backup"
  path             = %q
  credentials      = %s
  attributes_json  = jsonencode({ bucket = %q, folder = "tf-acc-test" })
  password         = %q
  keep_last        = 5
  enabled          = false
}
`, path, credID, bucket, password)
}

// testAccCheckCloudBackupDestroy verifies the cloud_backup is gone
// from the upstream after Terraform removes it. A green check here on
// a leaked resource is exactly the failure mode the static
// TestAcceptanceTestsHaveCheckDestroy invariant exists to flag — and
// this is the behavioral counterpart that actually verifies the
// removal happened.
func testAccCheckCloudBackupDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("cloud_backup ID not set on %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("cloud_backup ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetCloudBackup(ctx, id)
		if err == nil {
			return fmt.Errorf("cloud_backup %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of cloud_backup %d: %w", id, err)
		}
		return nil
	}
}
