package resources_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// TestAccZvol_refreservation covers GitHub #42: a thick zvol reads its
// automatic reservation back, a volsize change on it updates in place
// without an inconsistent result (ZFS recomputes the reservation), and
// refreservation = 0 thins it in place, never by replacement.
func TestAccZvol_refreservation(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-zvol-refres"
	resourceName := "truenas_zvol.test"
	atLeast := func(min int64) resource.CheckResourceAttrWithFunc {
		return func(v string) error {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("refreservation %q is not an integer: %w", v, err)
			}
			if n < min {
				return fmt.Errorf("refreservation = %d, want at least %d (thick)", n, min)
			}
			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckZvolDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccZvolConfigRefres(pool, name, 67108864, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(resourceName, "refreservation", atLeast(67108864)),
				),
			},
			{
				Config: testAccZvolConfigRefres(pool, name, 134217728, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "volsize", "134217728"),
					// 25.10 keeps ZFS's automatic reservation, which follows volsize.
					// TrueNAS 26 sets it to volsize at create (source LOCAL), so ZFS
					// leaves it alone on the resize: the 64 MiB from step 1 stays.
					resource.TestCheckResourceAttrWith(resourceName, "refreservation", func(v string) error {
						sv, err := zvolServerVersion()
						if err != nil {
							return err
						}
						if sv.AtLeast(26, 0) {
							if v != "67108864" {
								return fmt.Errorf("refreservation = %s on TrueNAS %s, want 67108864 (set at create, kept on resize)", v, sv)
							}
							return nil
						}
						return atLeast(134217728)(v)
					}),
				),
			},
			{
				Config: testAccZvolConfigRefres(pool, name, 134217728, "0"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "refreservation", "0"),
			},
			// A pinned non-zero reservation must survive a resize: the update
			// re-sends it so ZFS cannot leave a recomputed value behind.
			{
				Config: testAccZvolConfigRefres(pool, name, 134217728, "134217728"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "refreservation", "134217728"),
			},
			{
				Config: testAccZvolConfigRefres(pool, name, 268435456, "134217728"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "volsize", "268435456"),
					resource.TestCheckResourceAttr(resourceName, "refreservation", "134217728"),
				),
			},
		},
	})
}

func zvolServerVersion() (wsclient.ServerVersion, error) {
	c, err := acctest.Client()
	if err != nil {
		return wsclient.ServerVersion{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.ServerVersion(ctx)
}

func testAccZvolConfigRefres(pool, name string, volsize int64, refreservation string) string {
	line := ""
	if refreservation != "" {
		line = "  refreservation = " + refreservation + "\n"
	}
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_zvol" "test" {
  pool    = %q
  name    = %q
  volsize = %d
%s}
`, pool, name, volsize, line)
}
