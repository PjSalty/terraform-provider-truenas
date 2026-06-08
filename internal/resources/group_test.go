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

func TestAccGroup_basic(t *testing.T) {
	resourceName := "truenas_group.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfigBasic("tfaccgrp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tfaccgrp"),
					resource.TestCheckResourceAttr(resourceName, "smb", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "gid"),
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

func TestAccGroup_update(t *testing.T) {
	resourceName := "truenas_group.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfigWithSMB("tfaccgrpupd", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tfaccgrpupd"),
					resource.TestCheckResourceAttr(resourceName, "smb", "false"),
				),
			},
			// Update SMB in-place
			{
				Config: testAccGroupConfigWithSMB("tfaccgrpupd", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "smb", "true"),
				),
			},
		},
	})
}

func testAccCheckGroupDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("group ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("group ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetGroup(ctx, id)
		if err == nil {
			return fmt.Errorf("group %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of group %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckGroupExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetGroup(ctx, id); err != nil {
			return fmt.Errorf("group %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckGroupDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteGroup(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of group %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccGroup_disappears(t *testing.T) {
	resourceName := "truenas_group.test"
	name := fmt.Sprintf("tf-acc-grp-disappears-%d", acctest.ShortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfigBasic(name),
				Check:  testAccCheckGroupExists(resourceName),
			},
			{
				Config:             testAccGroupConfigBasic(name),
				Check:              testAccCheckGroupDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccGroupConfigBasic(name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_group" "test" {
  name = %q
}
`, name)
}

func testAccGroupConfigWithSMB(name string, smb bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_group" "test" {
  name = %q
  smb  = %t
}
`, name, smb)
}
