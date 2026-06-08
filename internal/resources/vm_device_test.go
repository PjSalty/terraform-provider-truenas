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

// testAccVMDeviceTargetVM returns the numeric ID of the VM used as the parent
// for device tests. The VM must already exist on the test server.
func testAccVMDeviceTargetVM() string {
	if v := os.Getenv("TRUENAS_TEST_VM_ID"); v != "" {
		return v
	}
	return "1"
}

func TestAccVMDevice_display(t *testing.T) {
	// Match the _disappears gate: VM device tests need an existing
	// parent VM that this suite does NOT create (VM-create is heavy
	// and the test pool may not have backing storage). Previously the
	// helper fell back to VM ID 1, which hard-fails on instances
	// where no VMs have been provisioned ("VM 1 does not exist").
	vmID := os.Getenv("TRUENAS_TEST_VM_ID")
	if vmID == "" {
		t.Skip("TRUENAS_TEST_VM_ID not set; VM device tests need a parent VM that the test does not create")
	}
	resourceName := "truenas_vm_device.test"

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
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("VM device ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetVMDevice(ctx, id)
		if err == nil {
			return fmt.Errorf("VM device %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of VM device %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckVMDeviceExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetVMDevice(ctx, id); err != nil {
			return fmt.Errorf("VM device %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckVMDeviceDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteVMDevice(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of VM device %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccVMDevice_disappears(t *testing.T) {
	vmID := os.Getenv("TRUENAS_TEST_VM_ID")
	if vmID == "" {
		t.Skip("TRUENAS_TEST_VM_ID not set; VM device tests need a parent VM that the test does not create")
	}
	resourceName := "truenas_vm_device.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMDeviceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccVMDeviceConfigDisplay(vmID, "1024x768"),
				Check:  testAccCheckVMDeviceExists(resourceName),
			},
			{
				Config:             testAccVMDeviceConfigDisplay(vmID, "1024x768"),
				Check:              testAccCheckVMDeviceDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
