package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertClasses_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_ALERTCLASSES") != "1" {
		t.Skip("TRUENAS_TEST_ALERTCLASSES=1 not set; skipping (alertclasses is a global singleton)")
	}

	resourceName := "truenas_alertclasses.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertClassesConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "alertclasses"),
					resource.TestCheckResourceAttr(resourceName, "classes.ZpoolCapacityNotice.level", "NOTICE"),
					resource.TestCheckResourceAttr(resourceName, "classes.ZpoolCapacityNotice.policy", "HOURLY"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "alertclasses",
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAlertClassesConfigBasic() string {
	return `
provider "truenas" {}

resource "truenas_alertclasses" "test" {
  classes = {
    ZpoolCapacityNotice = {
      level  = "NOTICE"
      policy = "HOURLY"
    }
  }
}
`
}
