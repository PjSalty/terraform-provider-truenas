package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMailConfigResource_basic — singleton: mail service configuration
// has a single instance and Delete is a no-op reset. Uses a dummy
// outgoing server so the test never sends real email.
func TestAccMailConfigResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_mail_config" "test" {
  fromemail      = "acctest@example.invalid"
  fromname       = "acctest"
  outgoingserver = "smtp.example.invalid"
  port           = 25
  security       = "PLAIN"
  smtp           = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_mail_config.test", "fromemail", "acctest@example.invalid"),
					resource.TestCheckResourceAttr("truenas_mail_config.test", "outgoingserver", "smtp.example.invalid"),
					resource.TestCheckResourceAttrSet("truenas_mail_config.test", "id"),
				),
			},
		},
	})
}

// TestAccMailConfigResource_update toggles the fromname on the
// singleton mail config. Third step restores an empty fromname so the
// shared test VM is left in a predictable state. fromemail is kept
// stable because the backend's Update function does not reset it to
// empty (see mail_config.go buildUpdateRequest).
func TestAccMailConfigResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	cfg := func(fromname string) string {
		return fmt.Sprintf(`
resource "truenas_mail_config" "test" {
  fromemail      = "acctest@example.invalid"
  fromname       = %q
  outgoingserver = "smtp.example.invalid"
  port           = 25
  security       = "PLAIN"
  smtp           = false
}
`, fromname)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("acctest-initial"),
				Check:  resource.TestCheckResourceAttr("truenas_mail_config.test", "fromname", "acctest-initial"),
			},
			{
				Config: cfg("acctest-updated"),
				Check:  resource.TestCheckResourceAttr("truenas_mail_config.test", "fromname", "acctest-updated"),
			},
			{
				// Restore the default (empty) fromname so the shared
				// test VM is left in a predictable state for subsequent
				// runs.
				Config: cfg(""),
				Check:  resource.TestCheckResourceAttr("truenas_mail_config.test", "fromname", ""),
			},
		},
	})
}
