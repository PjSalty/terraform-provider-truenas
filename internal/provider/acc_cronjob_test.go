package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCronJobResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_cronjob" "test" {
  user           = "root"
  command        = "/bin/true"
  description    = "acctest"
  enabled        = false
  schedule_minute = "0"
  schedule_hour   = "0"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "user", "root"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "/bin/true"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "enabled", "false"),
					resource.TestCheckResourceAttrSet("truenas_cronjob.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_cronjob.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccCronJobResource_update changes the description in place. The
// cron schedule fields and command are kept identical so only a
// true in-place update is exercised.
func TestAccCronJobResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	cfg := func(desc string) string {
		return fmt.Sprintf(`
resource "truenas_cronjob" "test" {
  user            = "root"
  command         = "/bin/true"
  description     = %q
  enabled         = false
  schedule_minute = "0"
  schedule_hour   = "0"
}
`, desc)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("acctest initial"),
				Check:  resource.TestCheckResourceAttr("truenas_cronjob.test", "description", "acctest initial"),
			},
			{
				Config: cfg("acctest updated"),
				Check:  resource.TestCheckResourceAttr("truenas_cronjob.test", "description", "acctest updated"),
			},
		},
	})
}

// TestAccCronJobResource_disappears deletes the cronjob out-of-band.
func TestAccCronJobResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_cronjob" "test" {
  user            = "root"
  command         = "/bin/true"
  description     = "acctest disappears"
  enabled         = false
  schedule_minute = "0"
  schedule_hour   = "0"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_cronjob.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_cronjob.test"]
						if !ok {
							return fmt.Errorf("truenas_cronjob.test not in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parse id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteCronJob(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
