package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// WebShare is new in TrueNAS 26.0. On 25.10 and older the
// sharing.webshare namespace does not exist at all, so these tests skip
// rather than fail: a red suite on a 25.10 box would say nothing about
// the provider.
func testAccWebsharePreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	if _, err := c.ListWebshares(ctx); err != nil {
		t.Skipf("WebShare unavailable on this server, skipping: %v", err)
	}
}

func TestAccWebshare_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := fmt.Sprintf("tf-acc-webshare-basic-%d", acctest.ShortSuffix())
	shareName := fmt.Sprintf("tf-acc-ws-%d", acctest.ShortSuffix())
	resourceName := "truenas_share_webshare.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccWebsharePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebshareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccWebshareConfig(pool, datasetName, shareName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", shareName),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "is_home_base", "false"),
					// dataset, relative_path and locked are NOT asserted here.
					// They are derived server-side and legitimately empty for
					// some paths: a live 26.0 box returned an empty dataset for
					// a share on a dataset mountpoint, and TestCheckResourceAttrSet
					// treats empty as unset. What matters is that they come back
					// KNOWN rather than null, so plans do not show them as
					// "(known after apply)" forever, and that is pinned by
					// TestWebshareResource_nullDerivedFieldsReadAsKnown.
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccWebshare_update(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := fmt.Sprintf("tf-acc-webshare-update-%d", acctest.ShortSuffix())
	shareName := fmt.Sprintf("tf-acc-ws-upd-%d", acctest.ShortSuffix())
	resourceName := "truenas_share_webshare.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccWebsharePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebshareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccWebshareConfig(pool, datasetName, shareName, true),
				Check:  resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
			},
			// Toggling enabled is the cheapest round trip that proves the
			// update path sends what it claims.
			{
				Config: testAccWebshareConfig(pool, datasetName, shareName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "name", shareName),
				),
			},
		},
	})
}

func testAccCheckWebshareDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("WebShare share ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("WebShare share ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetWebshare(ctx, id)
		if err == nil {
			return fmt.Errorf("WebShare share %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of WebShare share %d: %w", id, err)
		}
		return nil
	}
}

func testAccWebshareConfig(pool, datasetName, shareName string, enabled bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "webshare_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_webshare" "test" {
  name    = %q
  path    = truenas_dataset.webshare_parent.mount_point
  enabled = %t
}
`, pool, datasetName, shareName, enabled)
}

// A share deleted out of band must be detected and re-planned, not error
// on every subsequent run.
func TestAccWebshare_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := fmt.Sprintf("tf-acc-webshare-disap-%d", acctest.ShortSuffix())
	shareName := fmt.Sprintf("tf-acc-ws-dis-%d", acctest.ShortSuffix())
	resourceName := "truenas_share_webshare.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccWebsharePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebshareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccWebshareConfig(pool, datasetName, shareName, true),
				Check:  testAccCheckWebshareExists(resourceName),
			},
			{
				Config:             testAccWebshareConfig(pool, datasetName, shareName, true),
				Check:              testAccCheckWebshareDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckWebshareExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("WebShare share ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetWebshare(ctx, id); err != nil {
			return fmt.Errorf("WebShare share %d not found upstream: %w", id, err)
		}
		return nil
	}
}

// testAccCheckWebshareDisappears deletes the share behind Terraform's back,
// which is what the next step's non-empty plan is asserting on.
func testAccCheckWebshareDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("WebShare share ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		return c.DeleteWebshare(ctx, id)
	}
}
