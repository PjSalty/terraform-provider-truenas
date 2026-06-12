package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestAccVM_basic(t *testing.T) {
	resourceName := "truenas_vm.test"
	// SCALE 25.10 schema: VM names must be alphanumeric only — no hyphens.
	name := "tfacctestvm"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create + read
			{
				// memory is in MB on SCALE 25.10+ (validator: 20–4194304).
				// Earlier SCALE accepted bytes; 256 MB works on either.
				Config: testAccVMConfigBasic(name, "initial description", 256),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "initial description"),
					resource.TestCheckResourceAttr(resourceName, "memory", "256"),
					resource.TestCheckResourceAttr(resourceName, "vcpus", "1"),
					resource.TestCheckResourceAttr(resourceName, "cores", "1"),
					resource.TestCheckResourceAttr(resourceName, "threads", "1"),
					resource.TestCheckResourceAttr(resourceName, "bootloader", "UEFI"),
					resource.TestCheckResourceAttr(resourceName, "time", "UTC"),
				),
			},
			// Update description
			{
				Config: testAccVMConfigBasic(name, "updated description", 256),
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
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("VM ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetVM(ctx, id)
		if err == nil {
			return fmt.Errorf("VM %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of VM %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckVMExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetVM(ctx, id); err != nil {
			return fmt.Errorf("VM %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckVMDisappears(resourceName string) resource.TestCheckFunc {
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
		// Force=true so a running VM doesn't block the destroy; the
		// disappears test wants to model an operator who reaches into
		// the UI and does whatever it takes to make the VM go away.
		if err := c.DeleteVM(ctx, id, &truenas.VMDeleteOptions{Force: true}); err != nil {
			return fmt.Errorf("out-of-band delete of VM %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccVM_disappears(t *testing.T) {
	resourceName := "truenas_vm.test"
	// SCALE 25.10 VM name validator is alphanumeric-only; no hyphens.
	name := fmt.Sprintf("tfaccvmdisappears%d", acctest.ShortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVMDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccVMConfigBasic(name, "disappears test VM", 512),
				Check:  testAccCheckVMExists(resourceName),
			},
			{
				Config:             testAccVMConfigBasic(name, "disappears test VM", 512),
				Check:              testAccCheckVMDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
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
