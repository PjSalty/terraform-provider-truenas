package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccZvolResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestzv")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_zvol" "test" {
  pool    = "test"
  name    = %q
  volsize = 16777216
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_zvol.test", "name", name),
					resource.TestCheckResourceAttr("truenas_zvol.test", "pool", "test"),
					resource.TestCheckResourceAttr("truenas_zvol.test", "volsize", "16777216"),
					resource.TestCheckResourceAttrSet("truenas_zvol.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_zvol.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccZvolResource_disappears verifies the provider detects out-of-band
// deletion of a zvol and reports drift on the next plan.
func TestAccZvolResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestzv")
	fullID := "test/" + name
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_zvol" "test" {
  pool    = "test"
  name    = %q
  volsize = 16777216
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_zvol.test", "id", fullID),
					func(s *terraform.State) error {
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteZvol(ctx, fullID)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccZvolResource_update toggles an updatable attribute (comments)
// to prove the Update path functions end-to-end.
func TestAccZvolResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := randomName("acctestzv")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_zvol" "test" {
  pool     = "test"
  name     = %q
  volsize  = 16777216
  comments = "initial"
}
`, name),
				Check: resource.TestCheckResourceAttr("truenas_zvol.test", "comments", "initial"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_zvol" "test" {
  pool        = "test"
  name        = %q
  volsize     = 16777216
  comments    = "updated"
  compression = "LZ4"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_zvol.test", "comments", "updated"),
					resource.TestCheckResourceAttr("truenas_zvol.test", "compression", "LZ4"),
				),
			},
		},
	})
}
