package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		return nil
	}
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
