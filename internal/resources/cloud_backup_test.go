package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
