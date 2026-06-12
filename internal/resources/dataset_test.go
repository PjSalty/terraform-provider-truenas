package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/provider"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"

	tfproviderserver "github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the in-process provider for acceptance tests.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"truenas": tfproviderserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck validates that required environment variables are present.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("TRUENAS_URL"); v == "" {
		t.Fatal("TRUENAS_URL must be set for acceptance tests")
	}
	if v := os.Getenv("TRUENAS_API_KEY"); v == "" {
		t.Fatal("TRUENAS_API_KEY must be set for acceptance tests")
	}
}

// testAccDatasetPool returns the pool name used in acceptance tests.
func testAccDatasetPool() string {
	if v := os.Getenv("TRUENAS_TEST_POOL"); v != "" {
		return v
	}
	return "tank"
}

func TestAccDataset_basic(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-basic"
	resourceName := "truenas_dataset.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccDatasetConfigBasic(pool, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pool", pool),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "type", "FILESYSTEM"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "mount_point"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// parent_dataset is not returned by the API in a form that
				// always round-trips; skip it in import verification.
				ImportStateVerifyIgnore: []string{"parent_dataset"},
			},
		},
	})
}

func TestAccDataset_withCompression(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-compression"
	resourceName := "truenas_dataset.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetConfigWithCompression(pool, name, "LZ4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "compression", "LZ4"),
				),
			},
			// Update compression in-place (no replace required)
			{
				Config: testAccDatasetConfigWithCompression(pool, name, "ZSTD"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "compression", "ZSTD"),
				),
			},
		},
	})
}

func TestAccDataset_requiresReplaceOnPoolChange(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-replace"
	resourceName := "truenas_dataset.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetConfigBasic(pool, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pool", pool),
				),
			},
			// Changing the name triggers a destroy+recreate (RequiresReplace).
			{
				Config: testAccDatasetConfigBasic(pool, name+"-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name+"-renamed"),
				),
			},
		},
	})
}

// testAccCheckDatasetDestroy verifies the dataset no longer exists on
// the TrueNAS host after Terraform destroys it. The framework calls
// this *after* the destroy step has run, so a successful lookup here
// means the destroy was a no-op (or the upstream is leaking state).
//
// Returning nil for "not found" is the success path; we want the API
// to confirm the resource is gone.
func testAccCheckDatasetDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			// Resource never reached state, so there's nothing to
			// confirm against the API. Treat as pass.
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("dataset ID not set")
		}

		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client for destroy check: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()

		_, err = c.GetDataset(ctx, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("dataset %q still exists after terraform destroy", rs.Primary.ID)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking destroy of %q: %w", rs.Primary.ID, err)
		}
		// IsNotFound is the success path: dataset is gone.
		return nil
	}
}

// testAccCheckDatasetExists verifies the dataset is reachable on the
// TrueNAS host. Used as a positive guard before
// testAccCheckDatasetDisappears tears it down out of band; without
// this guard, a missing-from-the-start dataset would silently pass
// the disappears check on a vacuously-true outcome.
func testAccCheckDatasetExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("dataset ID not set on %s", resourceName)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client for exists check: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetDataset(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("dataset %q should exist but lookup failed: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

// testAccCheckDatasetDisappears deletes the dataset out of band via
// the API (bypassing Terraform), simulating an operator who clicks
// "delete" in the TrueNAS UI between two terraform plans. The next
// `terraform plan` MUST detect the missing resource and propose
// re-creation — the framework's TestStep with ExpectNonEmptyPlan:true
// asserts exactly that, and a passing test proves the resource's
// Read handler calls resp.State.RemoveResource(ctx) on 404 (the
// invariant checked statically by TestResourcesRemoveFromStateOnNotFound).
func testAccCheckDatasetDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("dataset ID not set on %s", resourceName)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client for disappears check: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteDataset(ctx, rs.Primary.ID); err != nil {
			return fmt.Errorf("out-of-band delete of dataset %q failed: %w", rs.Primary.ID, err)
		}
		return nil
	}
}

// TestAccDataset_disappears exercises the out-of-band-delete recovery
// path: provision a dataset through Terraform, delete it directly via
// the TrueNAS API behind Terraform's back, then verify that the next
// plan correctly notices the resource is gone and proposes
// re-creation (instead of producing the same update-in-place diff
// forever).
//
// This is the per-resource analogue of the standard `_disappears`
// pattern major providers use. The static-analysis counterpart is
// TestResourcesRemoveFromStateOnNotFound in internal/provider; that
// test guards the *code*, this one guards the *behavior*.
func TestAccDataset_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	name := "tf-acc-test-disappears"
	resourceName := "truenas_dataset.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDatasetDestroy(resourceName),
		Steps: []resource.TestStep{
			// Step 1: create the dataset and confirm both Terraform and
			// the upstream agree it exists.
			{
				Config: testAccDatasetConfigBasic(pool, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDatasetExists(resourceName),
				),
			},
			// Step 2: delete out of band, then immediately refresh.
			// The framework runs Check after Apply; we delete inside
			// Check and then assert the resulting plan is non-empty —
			// which it will only be if the Read handler properly
			// removed the resource from state on 404.
			{
				Config: testAccDatasetConfigBasic(pool, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDatasetDisappears(resourceName),
				),
				// A correct provider sees the missing dataset on the
				// next refresh, removes it from state, and the planner
				// proposes "create" again. ExpectNonEmptyPlan:true is
				// load-bearing — if it's false here, the test will
				// claim PASS on a state file that's silently broken.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDatasetConfigBasic(pool, name string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "test" {
  pool = %q
  name = %q
}
`, pool, name)
}

func testAccDatasetConfigWithCompression(pool, name, compression string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "test" {
  pool        = %q
  name        = %q
  compression = %q
}
`, pool, name, compression)
}
