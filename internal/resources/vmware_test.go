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
		CheckDestroy:             testAccCheckVMwareDestroy(resourceName),
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

func testAccCheckVMwareExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetVMware(ctx, id); err != nil {
			return fmt.Errorf("vmware %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckVMwareDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteVMware(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of vmware %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccVMware_disappears(t *testing.T) {
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
		CheckDestroy:             testAccCheckVMwareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccVMwareConfigBasic(host, user, pass, datastore, filesystem),
				Check:  testAccCheckVMwareExists(resourceName),
			},
			{
				Config:             testAccVMwareConfigBasic(host, user, pass, datastore, filesystem),
				Check:              testAccCheckVMwareDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// testAccCheckVMwareDestroy verifies the VMware integration is gone
// from the upstream after Terraform removes it.
func testAccCheckVMwareDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("vmware ID not set on %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("vmware ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetVMware(ctx, id)
		if err == nil {
			return fmt.Errorf("vmware integration %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of vmware %d: %w", id, err)
		}
		return nil
	}
}
