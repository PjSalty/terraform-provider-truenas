package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestAccISCSIExtent_fileType(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-iscsi-extent-file"
	resourceName := "truenas_iscsi_extent.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIExtentDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIExtentConfigFile(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-extent-file"),
					resource.TestCheckResourceAttr(resourceName, "type", "FILE"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "blocksize", "512"),
					resource.TestCheckResourceAttr(resourceName, "rpm", "SSD"),
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

func testAccCheckISCSIExtentDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI extent ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("iSCSI extent ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetISCSIExtent(ctx, id)
		if err == nil {
			return fmt.Errorf("iSCSI extent %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of iSCSI extent %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSIExtentExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetISCSIExtent(ctx, id); err != nil {
			return fmt.Errorf("iSCSI extent %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSIExtentDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteISCSIExtent(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of iSCSI extent %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccISCSIExtent_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-iscsiext-disappears"
	resourceName := "truenas_iscsi_extent.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSIExtentDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIExtentConfigFile(pool, datasetName),
				Check:  testAccCheckISCSIExtentExists(resourceName),
			},
			{
				Config:             testAccISCSIExtentConfigFile(pool, datasetName),
				Check:              testAccCheckISCSIExtentDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccISCSIExtentConfigFile(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "extent_parent" {
  pool = %q
  name = %q
}

resource "truenas_iscsi_extent" "test" {
  name     = "tf-acc-extent-file"
  type     = "FILE"
  path     = "${truenas_dataset.extent_parent.mount_point}/extent.img"
  filesize = 536870912
}
`, pool, datasetName)
}
