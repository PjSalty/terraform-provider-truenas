package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVMware_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_VMWARE") != "1" {
		t.Skip("TRUENAS_TEST_VMWARE=1 not set; skipping (requires a reachable VMware host)")
	}

	host := os.Getenv("TRUENAS_TEST_VMWARE_HOST")
	user := os.Getenv("TRUENAS_TEST_VMWARE_USER")
	pass := os.Getenv("TRUENAS_TEST_VMWARE_PASS")
	datastore := os.Getenv("TRUENAS_TEST_VMWARE_DATASTORE")
	filesystem := os.Getenv("TRUENAS_TEST_VMWARE_FS")
	if host == "" || user == "" || pass == "" || datastore == "" || filesystem == "" {
		t.Skip("required TRUENAS_TEST_VMWARE_* vars not set")
	}

	resourceName := "truenas_vmware.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVMwareConfigBasic(host, user, pass, datastore, filesystem),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "hostname", host),
					resource.TestCheckResourceAttr(resourceName, "username", user),
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

func testAccVMwareConfigBasic(host, user, pass, datastore, fs string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_vmware" "test" {
  hostname   = %q
  username   = %q
  password   = %q
  datastore  = %q
  filesystem = %q
}
`, host, user, pass, datastore, fs)
}
