package resources_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccScrubTaskCleanup deletes the existing scrub task for pool 1 if it exists,
// so the acceptance test can create a new one.
func testAccScrubTaskCleanup(t *testing.T) {
	t.Helper()
	url := os.Getenv("TRUENAS_URL")
	apiKey := os.Getenv("TRUENAS_API_KEY")
	if url == "" || apiKey == "" {
		return
	}

	req, _ := http.NewRequest("DELETE", url+"/api/v2.0/pool/scrub/id/1", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return // best-effort cleanup
	}
	resp.Body.Close()
}

func TestAccScrubTask_basic(t *testing.T) {
	resourceName := "truenas_scrub_task.test"

	// Delete existing scrub task so we can create a new one
	testAccScrubTaskCleanup(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckScrubTaskDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccScrubTaskConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "threshold", "35"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "schedule_dow", "7"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckScrubTaskDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("scrub task ID not set")
		}
		return nil
	}
}

func testAccScrubTaskConfigBasic() string {
	return `
provider "truenas" {}

data "truenas_pool" "test" {
  name = "test"
}

resource "truenas_scrub_task" "test" {
  pool         = data.truenas_pool.test.id
  threshold    = 35
  schedule_dow = "7"
}
`
}
