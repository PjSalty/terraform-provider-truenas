package resources_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccVMDataSource_basic reads a specific VM by id. Env-gated
// because creating a VM via the resource path requires a dedicated
// virt pool (TRUENAS_TEST_VM_ID points at an already-provisioned
// VM so the datasource Read path is exercised without the heavy
// resource create).
func TestAccVMDataSource_basic(t *testing.T) {
	id := os.Getenv("TRUENAS_TEST_VM_ID")
	if id == "" {
		t.Skip("set TRUENAS_TEST_VM_ID=<vm-id> to run the VM datasource acc test")
	}
	dataSourceName := "data.truenas_vm.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_vm" "test" {
  id = %s
}
`, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", id),
					resource.TestCheckResourceAttrSet(dataSourceName, "name"),
				),
			},
		},
	})
}

func TestAccVMDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

data "truenas_vm" "test" {
  id = %d
}
`, 99999),
				ExpectError: regexp.MustCompile(`(?i)not found|404|reading`),
			},
		},
	})
}
