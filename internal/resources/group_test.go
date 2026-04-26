package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		return nil
	}
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
