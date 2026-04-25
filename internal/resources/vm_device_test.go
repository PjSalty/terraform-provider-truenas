package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccVMDeviceTargetVM returns the numeric ID of the VM used as the parent
// for device tests. The VM must already exist on the test server.
func testAccVMDeviceTargetVM() string {
	if v := os.Getenv("TRUENAS_TEST_VM_ID"); v != "" {
		return v
	}
	return "1"
}

func TestAccVMDevice_display(t *testing.T) {
	resourceName := "truenas_vm_device.test"
	vmID := testAccVMDeviceTargetVM()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMDeviceDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create a DISPLAY device attached to the target VM.
			{
				Config: testAccVMDeviceConfigDisplay(vmID, "1024x768"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "vm", vmID),
					resource.TestCheckResourceAttr(resourceName, "dtype", "DISPLAY"),
					resource.TestCheckResourceAttr(resourceName, "attributes.resolution", "1024x768"),
				),
			},
			// Update resolution
			{
				Config: testAccVMDeviceConfigDisplay(vmID, "1280x720"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "attributes.resolution", "1280x720"),
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

func testAccCheckVMDeviceDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("VM device ID not set")
		}
		return nil
	}
}

func testAccVMDeviceConfigDisplay(vmID, resolution string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_vm_device" "test" {
  vm    = %s
  dtype = "DISPLAY"

  attributes = {
    resolution = %q
    bind       = "0.0.0.0"
    web        = "true"
    wait       = "false"
  }
}
`, vmID, resolution)
}
