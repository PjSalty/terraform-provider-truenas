package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// vmDeviceConfig wraps a test VM and a DISPLAY device attached to it.
// DISPLAY avoids the need for a backing disk or real NIC bridge and is
// accepted on any TrueNAS SCALE host.
func vmDeviceConfig(vmName string, resolution string) string {
	return fmt.Sprintf(`
resource "truenas_vm" "test" {
  name   = %q
  vcpus  = 1
  cores  = 1
  memory = 128
}

resource "truenas_vm_device" "test" {
  vm    = tonumber(truenas_vm.test.id)
  dtype = "DISPLAY"
  attributes = {
    resolution = %q
    bind       = "127.0.0.1"
    # SCALE 25.10 requires DISPLAY ports >= 5900 (VNC range) and a
    # non-empty password. The password here is dummy — the test VM
    # isn't reachable externally and the VM is never actually started.
    port       = "5900"
    web        = "true"
    password   = "acctest-display-pw"
  }
}
`, vmName, resolution)
}

func TestAccVMDeviceResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	vmName := fmt.Sprintf("acctvmd%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: vmDeviceConfig(vmName, "1024x768"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_vm_device.test", "dtype", "DISPLAY"),
					resource.TestCheckResourceAttr("truenas_vm_device.test", "attributes.resolution", "1024x768"),
					resource.TestCheckResourceAttrSet("truenas_vm_device.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_vm_device.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The vm_device Import path can't infer the user's
				// original attribute set, so skip round-trip diff on the
				// attributes map and vm reference.
				ImportStateVerifyIgnore: []string{"attributes", "vm"},
			},
		},
	})
}

// TestAccVMDeviceResource_update changes the resolution attribute to
// verify PUT /vm/device without replacement.
func TestAccVMDeviceResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	vmName := fmt.Sprintf("acctvmdu%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: vmDeviceConfig(vmName, "1024x768"),
				Check:  resource.TestCheckResourceAttr("truenas_vm_device.test", "attributes.resolution", "1024x768"),
			},
			{
				Config: vmDeviceConfig(vmName, "1280x1024"),
				Check:  resource.TestCheckResourceAttr("truenas_vm_device.test", "attributes.resolution", "1280x1024"),
			},
		},
	})
}

func TestAccVMDeviceResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	vmName := fmt.Sprintf("acctvmdd%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: vmDeviceConfig(vmName, "1024x768"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_vm_device.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_vm_device.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parsing id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteVMDevice(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
