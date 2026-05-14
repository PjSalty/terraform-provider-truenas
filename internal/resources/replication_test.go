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

func TestAccReplication_local(t *testing.T) {
	pool := testAccDatasetPool()
	resourceName := "truenas_replication.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReplicationDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfigLocal(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-test-repl"),
					resource.TestCheckResourceAttr(resourceName, "direction", "PUSH"),
					resource.TestCheckResourceAttr(resourceName, "transport", "LOCAL"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "retention_policy", "SOURCE"),
				),
			},
		},
	})
}

func testAccCheckReplicationDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("replication ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("replication ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetReplication(ctx, id)
		if err == nil {
			return fmt.Errorf("replication %d still exists upstream after Terraform removed it", id)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of replication %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckReplicationExists(resourceName string) resource.TestCheckFunc {
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
		if _, err := c.GetReplication(ctx, id); err != nil {
			return fmt.Errorf("replication %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckReplicationDisappears(resourceName string) resource.TestCheckFunc {
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
		if err := c.DeleteReplication(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of replication %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccReplication_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	resourceName := "truenas_replication.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckReplicationDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfigLocal(pool),
				Check:  testAccCheckReplicationExists(resourceName),
			},
			{
				Config:             testAccReplicationConfigLocal(pool),
				Check:              testAccCheckReplicationDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccReplicationConfigLocal(pool string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "repl_src" {
  pool = %q
  name = "tf-acc-repl-src"
}

resource "truenas_dataset" "repl_dst" {
  pool = %q
  name = "tf-acc-repl-dst"
}

resource "truenas_replication" "test" {
  name             = "tf-acc-test-repl"
  direction        = "PUSH"
  transport        = "LOCAL"
  source_datasets  = ["${truenas_dataset.repl_src.pool}/${truenas_dataset.repl_src.name}"]
  target_dataset   = "${truenas_dataset.repl_dst.pool}/${truenas_dataset.repl_dst.name}"
  retention_policy = "SOURCE"
  auto             = false
  enabled          = true

  also_include_naming_schema = ["auto-%%Y-%%m-%%d_%%H-%%M"]
}
`, pool, pool)
}
