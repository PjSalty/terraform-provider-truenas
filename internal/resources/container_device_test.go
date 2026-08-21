package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// Container devices are new in TrueNAS 26.0. On 25.10 and older the
// container.device namespace does not exist, so these skip rather than
// fail: a red suite on a 25.10 box would say nothing about the provider.
func testAccContainerDevicePreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	if _, err := c.ListContainerDevices(ctx); err != nil {
		t.Skipf("container devices unavailable on this server, skipping: %v", err)
	}
}

// testAccContainerDeviceBridge returns a bridge a NIC device can attach
// to, or skips. Bridge names differ per host, so hardcoding one would
// fail everywhere but the box it was written on.
func testAccContainerDeviceBridge(t *testing.T) string {
	t.Helper()
	if !acctestEnabled() {
		return "placeholder"
	}
	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	choices, err := c.GetContainerNICAttachChoices(ctx)
	if err != nil {
		t.Skipf("NIC attach choices unavailable, skipping: %v", err)
	}
	if len(choices.Bridge) == 0 {
		t.Skip("no bridge available for NIC attachment, skipping")
	}
	return choices.Bridge[0]
}

func TestAccContainerDevice_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ct-dev-fs-%d", acctest.ShortSuffix())
	resourceName := "truenas_container_device.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerDevicePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerDeviceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccContainerDeviceFilesystemConfig(t, name, "/srv/data"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "filesystem.target", "/srv/data"),
					resource.TestCheckResourceAttrSet(resourceName, "filesystem.source"),
					// The three blocks that were not set must read back as
					// absent, not as empty objects.
					resource.TestCheckNoResourceAttr(resourceName, "gpu.gpu_type"),
					resource.TestCheckNoResourceAttr(resourceName, "nic.type"),
					resource.TestCheckNoResourceAttr(resourceName, "usb.device"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

// Changing a device attribute replaces the device: TrueNAS validates the
// attributes against the shape chosen at creation.
func TestAccContainerDevice_update(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ct-dev-upd-%d", acctest.ShortSuffix())
	resourceName := "truenas_container_device.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerDevicePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerDeviceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccContainerDeviceFilesystemConfig(t, name, "/srv/data"),
				Check:  resource.TestCheckResourceAttr(resourceName, "filesystem.target", "/srv/data"),
			},
			{
				Config: testAccContainerDeviceFilesystemConfig(t, name, "/srv/other"),
				Check:  resource.TestCheckResourceAttr(resourceName, "filesystem.target", "/srv/other"),
			},
		},
	})
}

// A NIC with no MAC set must keep the one TrueNAS generates, rather than
// planning a new one on every run.
func TestAccContainerDevice_nicGeneratedMACIsStable(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ct-dev-nic-%d", acctest.ShortSuffix())
	bridge := testAccContainerDeviceBridge(t)
	resourceName := "truenas_container_device.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerDevicePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerDeviceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccContainerDeviceNICConfig(t, name, bridge),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "nic.type", "VIRTIO"),
					resource.TestCheckResourceAttr(resourceName, "nic.nic_attach", bridge),
					resource.TestCheckResourceAttrSet(resourceName, "nic.mac"),
				),
			},
			{
				Config:             testAccContainerDeviceNICConfig(t, name, bridge),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// A device deleted out of band must be detected and re-planned, not error
// on every subsequent run.
func TestAccContainerDevice_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ct-dev-dis-%d", acctest.ShortSuffix())
	resourceName := "truenas_container_device.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerDevicePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerDeviceDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccContainerDeviceFilesystemConfig(t, name, "/srv/data"),
				Check:  testAccCheckContainerDeviceExists(resourceName),
			},
			{
				Config:             testAccContainerDeviceFilesystemConfig(t, name, "/srv/data"),
				Check:              testAccCheckContainerDeviceDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccContainerDeviceContainerConfig(t *testing.T, name string) string {
	t.Helper()
	imageName, imageVersion := testAccContainerImage(t)
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_container" "parent" {
  name = %q
  pool = %q

  image = {
    name    = %q
    version = %q
  }
}
`, name, testAccDatasetPool(), imageName, imageVersion)
}

func testAccContainerDeviceFilesystemConfig(t *testing.T, name, target string) string {
	t.Helper()
	// The pool mountpoint always exists and is always under /mnt, which is
	// what TrueNAS requires of a bind-mount source.
	return testAccContainerDeviceContainerConfig(t, name) + fmt.Sprintf(`
resource "truenas_container_device" "test" {
  container = truenas_container.parent.id

  filesystem = {
    source = "/mnt/%s"
    target = %q
  }
}
`, testAccDatasetPool(), target)
}

func testAccContainerDeviceNICConfig(t *testing.T, name, bridge string) string {
	t.Helper()
	return testAccContainerDeviceContainerConfig(t, name) + fmt.Sprintf(`
resource "truenas_container_device" "test" {
  container = truenas_container.parent.id

  nic = {
    type       = "VIRTIO"
    nic_attach = %q
  }
}
`, bridge)
}

func testAccContainerDeviceID(s *terraform.State, resourceName string) (int, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return 0, fmt.Errorf("not found in state: %s", resourceName)
	}
	id, err := strconv.Atoi(rs.Primary.ID)
	if err != nil {
		return 0, fmt.Errorf("container device ID %q is not numeric: %w", rs.Primary.ID, err)
	}
	return id, nil
}

func testAccCheckContainerDeviceDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if _, ok := s.RootModule().Resources[resourceName]; !ok {
			return nil
		}
		id, err := testAccContainerDeviceID(s, resourceName)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetContainerDevice(ctx, id)
		if err == nil {
			return fmt.Errorf("container device %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of container device %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckContainerDeviceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccContainerDeviceID(s, resourceName)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetContainerDevice(ctx, id); err != nil {
			return fmt.Errorf("container device %d not found upstream: %w", id, err)
		}
		return nil
	}
}

// testAccCheckContainerDeviceDisappears deletes the device behind
// Terraform's back, which is what the next step's non-empty plan asserts.
func testAccCheckContainerDeviceDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccContainerDeviceID(s, resourceName)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		return c.DeleteContainerDevice(ctx, id, nil)
	}
}
