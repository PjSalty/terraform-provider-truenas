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

func TestAccUser_basic(t *testing.T) {
	resourceName := "truenas_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigBasic("tfacctest", "TF Acc Test User"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", "tfacctest"),
					resource.TestCheckResourceAttr(resourceName, "full_name", "TF Acc Test User"),
					resource.TestCheckResourceAttr(resourceName, "locked", "false"),
					resource.TestCheckResourceAttr(resourceName, "smb", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "uid"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Password is write-only and not returned by the API.
				// group_create is only used during creation.
				ImportStateVerifyIgnore: []string{"password", "group_create"},
			},
		},
	})
}

func TestAccUser_update(t *testing.T) {
	resourceName := "truenas_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigBasic("tfaccupdate", "TF Acc Before"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "full_name", "TF Acc Before"),
				),
			},
			// Update full_name in-place
			{
				Config: testAccUserConfigBasic("tfaccupdate", "TF Acc After"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "full_name", "TF Acc After"),
				),
			},
		},
	})
}

func testAccCheckUserDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("user ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("user ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetUser(ctx, id)
		if err == nil {
			return fmt.Errorf("user %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of user %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckUserExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetUser(ctx, id); err != nil {
			return fmt.Errorf("user %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckUserDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteUser(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of user %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccUser_disappears(t *testing.T) {
	resourceName := "truenas_user.test"
	username := fmt.Sprintf("tf-acc-user-disappears-%d", acctest.ShortSuffix())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigBasic(username, "Disappears Test User"),
				Check:  testAccCheckUserExists(resourceName),
			},
			{
				Config:             testAccUserConfigBasic(username, "Disappears Test User"),
				Check:              testAccCheckUserDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccUserConfigBasic(username, fullName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_user" "test" {
  username  = %q
  full_name = %q
  password  = "TestP@ssw0rd123!"
}
`, username, fullName)
}
