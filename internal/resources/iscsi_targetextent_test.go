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

func TestAccISCSITargetExtent_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-iscsi-te"
	resourceName := "truenas_iscsi_targetextent.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSITargetExtentDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetExtentConfigBasic(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "target"),
					resource.TestCheckResourceAttrSet(resourceName, "extent"),
					resource.TestCheckResourceAttrSet(resourceName, "lunid"),
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

func testAccCheckISCSITargetExtentDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("iSCSI target-extent ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("iSCSI target-extent ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetISCSITargetExtent(ctx, id)
		if err == nil {
			return fmt.Errorf("iSCSI target-extent %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of iSCSI target-extent %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSITargetExtentExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetISCSITargetExtent(ctx, id); err != nil {
			return fmt.Errorf("iSCSI target-extent %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckISCSITargetExtentDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteISCSITargetExtent(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of iSCSI target-extent %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccISCSITargetExtent_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-tgtext-disappears"
	resourceName := "truenas_iscsi_targetextent.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckISCSITargetExtentDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetExtentConfigBasic(pool, datasetName),
				Check:  testAccCheckISCSITargetExtentExists(resourceName),
			},
			{
				Config:             testAccISCSITargetExtentConfigBasic(pool, datasetName),
				Check:              testAccCheckISCSITargetExtentDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccISCSITargetExtentConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "te_parent" {
  pool = %q
  name = %q
}

resource "truenas_iscsi_extent" "te_extent" {
  name     = "tf-acc-te-extent"
  type     = "FILE"
  path     = "${truenas_dataset.te_parent.mount_point}/te-extent.img"
  filesize = 536870912
}

resource "truenas_iscsi_target" "te_target" {
  name = "tf-acc-te-target"
  mode = "ISCSI"
}

resource "truenas_iscsi_targetextent" "test" {
  target = truenas_iscsi_target.te_target.id
  extent = truenas_iscsi_extent.te_extent.id
}
`, pool, datasetName)
}
