package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceResource_basic manages the ftp service and verifies it
// can be imported. We pick `ftp` because it's a benign, always-present
// TrueNAS service that isn't used by any other acceptance test. There
// is no `_disappears` variant for this resource — services cannot be
// deleted, only disabled — so a separate basic + update pair is the
// right coverage.
func TestAccServiceResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_service" "test" {
  service = "ftp"
  enable  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_service.test", "service", "ftp"),
					resource.TestCheckResourceAttr("truenas_service.test", "enable", "false"),
					resource.TestCheckResourceAttrSet("truenas_service.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccServiceResource_update toggles the `enable` flag on the ftp
// service — this is the only meaningful updatable attribute on the
// resource.
func TestAccServiceResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_service" "test" {
  service = "ftp"
  enable  = false
}
`,
				Check: resource.TestCheckResourceAttr("truenas_service.test", "enable", "false"),
			},
			{
				Config: `
resource "truenas_service" "test" {
  service = "ftp"
  enable  = true
}
`,
				Check: resource.TestCheckResourceAttr("truenas_service.test", "enable", "true"),
			},
		},
	})
}
