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

func TestAccTunable_basic(t *testing.T) {
	resourceName := "truenas_tunable.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTunableDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccTunableConfigBasic("SYSCTL", "net.ipv4.ip_forward", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "SYSCTL"),
					resource.TestCheckResourceAttr(resourceName, "var", "net.ipv4.ip_forward"),
					resource.TestCheckResourceAttr(resourceName, "value", "1"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
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

func TestAccTunable_update(t *testing.T) {
	resourceName := "truenas_tunable.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTunableDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccTunableConfigWithComment("SYSCTL", "net.ipv4.tcp_syncookies", "1", "Enable syncookies"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "value", "1"),
					resource.TestCheckResourceAttr(resourceName, "comment", "Enable syncookies"),
				),
			},
			// Update value and comment in-place
			{
				Config: testAccTunableConfigWithComment("SYSCTL", "net.ipv4.tcp_syncookies", "0", "Disable syncookies"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "value", "0"),
					resource.TestCheckResourceAttr(resourceName, "comment", "Disable syncookies"),
				),
			},
		},
	})
}

func testAccCheckTunableDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("tunable ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("tunable ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetTunable(ctx, id)
		if err == nil {
			return fmt.Errorf("tunable %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of tunable %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckTunableExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetTunable(ctx, id); err != nil {
			return fmt.Errorf("tunable %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckTunableDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteTunable(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of tunable %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccTunable_disappears(t *testing.T) {
	resourceName := "truenas_tunable.test"
	// SYSCTL tunable var must be a real /proc/sys path on SCALE 25.10+;
	// the API silently fails Create on unknown names with a generic
	// "tunable not found after creation" error. `kernel.acct` is a
	// benign BSD-process-accounting sysctl that's safe to set to 0.
	// We don't need a per-run-unique var name here because the
	// _disappears test out-of-band deletes the row before the next
	// step plans, so subsequent runs don't collide.
	varName := "kernel.acct"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTunableDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccTunableConfigBasic("SYSCTL", varName, "1"),
				Check:  testAccCheckTunableExists(resourceName),
			},
			{
				Config:             testAccTunableConfigBasic("SYSCTL", varName, "1"),
				Check:              testAccCheckTunableDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccTunableConfigBasic(tunableType, varName, value string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_tunable" "test" {
  type  = %q
  var   = %q
  value = %q
}
`, tunableType, varName, value)
}

func testAccTunableConfigWithComment(tunableType, varName, value, comment string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_tunable" "test" {
  type    = %q
  var     = %q
  value   = %q
  comment = %q
}
`, tunableType, varName, value, comment)
}
