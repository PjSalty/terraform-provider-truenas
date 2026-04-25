package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// nvmetNamespaceConfig builds an HCL config that creates a subsystem
// and a FILE-backed namespace inside it. Using a FILE backing avoids
// the need for a zvol/pool prereq — TrueNAS will create the file on
// demand in /var/run during testing.
func nvmetNamespaceConfig(subsysName, devicePath string, enabled bool) string {
	return fmt.Sprintf(`
resource "truenas_nvmet_subsys" "test" {
  name = %q
}

resource "truenas_nvmet_namespace" "test" {
  subsys_id    = tonumber(truenas_nvmet_subsys.test.id)
  device_type  = "FILE"
  device_path  = %q
  filesize     = 1048576
  enabled      = %t
}
`, subsysName, devicePath, enabled)
}

func TestAccNVMetNamespaceResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	subsysName := fmt.Sprintf("acctns%d", shortSuffix())
	devicePath := fmt.Sprintf("/mnt/test/acctestns%d.img", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetNamespaceConfig(subsysName, devicePath, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "device_type", "FILE"),
					resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "device_path", devicePath),
					resource.TestCheckResourceAttrSet("truenas_nvmet_namespace.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_namespace.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNVMetNamespaceResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	subsysName := fmt.Sprintf("acctns%d", shortSuffix())
	devicePath := fmt.Sprintf("/mnt/test/acctestnsu%d.img", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetNamespaceConfig(subsysName, devicePath, true),
				Check:  resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "enabled", "true"),
			},
			{
				Config: nvmetNamespaceConfig(subsysName, devicePath, false),
				Check:  resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "enabled", "false"),
			},
		},
	})
}

func TestAccNVMetNamespaceResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	subsysName := fmt.Sprintf("acctns%d", shortSuffix())
	devicePath := fmt.Sprintf("/mnt/test/acctestnsd%d.img", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: nvmetNamespaceConfig(subsysName, devicePath, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_namespace.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_nvmet_namespace.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parsing id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteNVMetNamespace(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
