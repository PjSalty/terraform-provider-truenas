package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccISCSIInitiatorDataSource_basic creates an iSCSI initiator
// group via the resource path and reads it via the datasource. The
// initiator group is a singleton-style record per allowed-host
// configuration; this validates the most common round-trip the
// operator hits when wiring an iSCSI workload.
func TestAccISCSIInitiatorDataSource_basic(t *testing.T) {
	dataSourceName := "data.truenas_iscsi_initiator.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

resource "truenas_iscsi_initiator" "fixture" {
  comment    = "tf-acc-initiator-ds"
  initiators = ["iqn.2026-06.com.example:test-host"]
}

data "truenas_iscsi_initiator" "test" {
  id = truenas_iscsi_initiator.fixture.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "comment", "tf-acc-initiator-ds"),
					resource.TestCheckResourceAttr(dataSourceName, "initiators.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "initiators.0", "iqn.2026-06.com.example:test-host"),
				),
			},
		},
	})
}

func TestAccISCSIInitiatorDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_iscsi_initiator" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
