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

func TestAccSMBShare_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smb-basic"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccSMBShareConfigBasic(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-smb-basic"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "browsable", "true"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
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

func TestAccSMBShare_update(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smb-update"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create with readonly false
			{
				Config: testAccSMBShareConfigReadOnly(pool, datasetName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
					resource.TestCheckResourceAttr(resourceName, "comment", "acc test share"),
				),
			},
			// Update: set readonly true
			{
				Config: testAccSMBShareConfigReadOnly(pool, datasetName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "readonly", "true"),
				),
			},
		},
	})
}

func testAccCheckSMBShareDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("SMB share ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("SMB share ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetSMBShare(ctx, id)
		if err == nil {
			return fmt.Errorf("SMB share %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of SMB share %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckSMBShareExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetSMBShare(ctx, id); err != nil {
			return fmt.Errorf("SMB share %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckSMBShareDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteSMBShare(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of SMB share %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccSMBShare_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smbshare-disappears"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccSMBShareConfigBasic(pool, datasetName),
				Check:  testAccCheckSMBShareExists(resourceName),
			},
			{
				Config:             testAccSMBShareConfigBasic(pool, datasetName),
				Check:              testAccCheckSMBShareDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccSMBShareConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "smb_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_smb" "test" {
  path = truenas_dataset.smb_parent.mount_point
  name = %q
}
`, pool, datasetName, datasetName)
}

func testAccSMBShareConfigReadOnly(pool, datasetName string, readOnly bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "smb_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_smb" "test" {
  path     = truenas_dataset.smb_parent.mount_point
  name     = %q
  readonly = %t
  comment  = "acc test share"
}
`, pool, datasetName, datasetName, readOnly)
}
