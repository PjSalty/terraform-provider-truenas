package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccVM_basic(t *testing.T) {
	resourceName := "truenas_vm.test"
	name := "tf-acc-test-vm"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create + read
			{
				Config: testAccVMConfigBasic(name, "initial description", 1073741824),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "initial description"),
					resource.TestCheckResourceAttr(resourceName, "memory", "1073741824"),
					resource.TestCheckResourceAttr(resourceName, "vcpus", "1"),
					resource.TestCheckResourceAttr(resourceName, "cores", "1"),
					resource.TestCheckResourceAttr(resourceName, "threads", "1"),
					resource.TestCheckResourceAttr(resourceName, "bootloader", "UEFI"),
					resource.TestCheckResourceAttr(resourceName, "time", "UTC"),
				),
			},
			// Update description
			{
				Config: testAccVMConfigBasic(name, "updated description", 1073741824),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "updated description"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"status"},
			},
		},
	})
}

func testAccCheckVMDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("VM ID not set")
		}
		return nil
	}
}

func testAccVMConfigBasic(name, description string, memory int64) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_vm" "test" {
  name        = %q
  description = %q
  memory      = %d
  vcpus       = 1
  cores       = 1
  threads     = 1
  bootloader  = "UEFI"
  time        = "UTC"
  autostart   = false
}
`, name, description, memory)
}
