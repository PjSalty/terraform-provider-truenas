package resources_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// Containers are new in TrueNAS 26.0. On 25.10 and older the container
// namespace does not exist at all, so these skip rather than fail: a red
// suite on a 25.10 box would say nothing about the provider.
func testAccContainerPreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	if _, err := c.ListContainers(ctx); err != nil {
		t.Skipf("containers unavailable on this server, skipping: %v", err)
	}
}

// testAccContainerImage resolves an image name and version from the live
// registry.
//
// Versions are datestamped and the registry keeps only the last few, so a
// hardcoded version stops existing within days and the whole suite starts
// failing for a reason that has nothing to do with the provider. Alpine is
// picked because it is the smallest image on offer, which keeps the
// create job (an image pull) short.
// acctestEnabled reports whether the acceptance suite is actually running.
// Config strings are built as arguments to resource.Test, so a helper that
// reaches the API runs even on a plain `go test` where resource.Test would
// skip.
func acctestEnabled() bool { return os.Getenv("TF_ACC") != "" }

func testAccContainerImage(t *testing.T) (string, string) {
	t.Helper()

	// Config strings are built as arguments to resource.Test, so this runs
	// even on a plain `go test` where resource.Test would skip. Without
	// this guard the unit run tries to reach a TrueNAS that is not
	// configured and fails three tests that were never meant to execute.
	if !acctestEnabled() {
		return "placeholder", "placeholder"
	}

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()

	raw, err := c.Call(ctx, "container.image.query_registry", nil,
		wsclient.CallOptions{Read: true, Idempotent: true})
	if err != nil {
		t.Skipf("image registry unreachable, skipping: %v", err)
	}

	var images []struct {
		Name     string `json:"name"`
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(raw, &images); err != nil {
		t.Fatalf("parsing image registry: %v", err)
	}
	for _, img := range images {
		if img.Name == "alpine:3.21:amd64:default" && len(img.Versions) > 0 {
			// The registry lists versions oldest-first, so the last entry
			// is the one least likely to age out mid-suite.
			return img.Name, img.Versions[len(img.Versions)-1].Version
		}
	}
	t.Skip("no alpine image in the registry, skipping")
	return "", ""
}

func TestAccContainer_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ct-%d", acctest.ShortSuffix())
	resourceName := "truenas_container.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccContainerConfig(t, name, "created by the acceptance suite", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "autostart", "true"),
					// Server-side defaults must land in state, or the next
					// plan shows a diff against values nobody set.
					resource.TestCheckResourceAttr(resourceName, "time", "LOCAL"),
					resource.TestCheckResourceAttr(resourceName, "init", "/sbin/init"),
					resource.TestCheckResourceAttr(resourceName, "shutdown_timeout", "90"),
					resource.TestCheckResourceAttr(resourceName, "capabilities_policy", "DEFAULT"),
					// Derived server-side.
					resource.TestCheckResourceAttrSet(resourceName, "uuid"),
					resource.TestCheckResourceAttrSet(resourceName, "dataset"),
					resource.TestCheckResourceAttrSet(resourceName, "status.state"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// image and pool are create-only: both are excluded from
				// the upstream entry, so an imported container reports
				// neither and a verify would compare them against nothing.
				ImportStateVerifyIgnore: []string{"image", "pool", "timeouts"},
			},
		},
	})
}

func TestAccContainer_update(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ct-upd-%d", acctest.ShortSuffix())
	resourceName := "truenas_container.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccContainerConfig(t, name, "before", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "before"),
					resource.TestCheckResourceAttr(resourceName, "autostart", "true"),
				),
			},
			// Both fields are on the update model upstream, so this proves
			// the update path sends what it claims without triggering the
			// RequiresReplace attributes.
			{
				Config: testAccContainerConfig(t, name, "after", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "after"),
					resource.TestCheckResourceAttr(resourceName, "autostart", "false"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
		},
	})
}

// A container deleted out of band must be detected and re-planned, not
// error on every subsequent run.
func TestAccContainer_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ct-dis-%d", acctest.ShortSuffix())
	resourceName := "truenas_container.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContainerDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccContainerConfig(t, name, "disappears", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: testAccCheckContainerExists(resourceName),
			},
			{
				Config:             testAccContainerConfig(t, name, "disappears", true),
				Check:              testAccCheckContainerDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccContainerConfig(t *testing.T, name, description string, autostart bool) string {
	t.Helper()
	imageName, imageVersion := testAccContainerImage(t)
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_container" "test" {
  name        = %q
  description = %q
  autostart   = %t
  pool        = %q

  image = {
    name    = %q
    version = %q
  }
}
`, name, description, autostart, testAccDatasetPool(), imageName, imageVersion)
}

func testAccContainerID(s *terraform.State, resourceName string) (int, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return 0, fmt.Errorf("not found in state: %s", resourceName)
	}
	if rs.Primary.ID == "" {
		return 0, fmt.Errorf("container ID not set")
	}
	id, err := strconv.Atoi(rs.Primary.ID)
	if err != nil {
		return 0, fmt.Errorf("container ID %q is not numeric: %w", rs.Primary.ID, err)
	}
	return id, nil
}

func testAccCheckContainerDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if _, ok := s.RootModule().Resources[resourceName]; !ok {
			return nil
		}
		id, err := testAccContainerID(s, resourceName)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetContainer(ctx, id)
		if err == nil {
			return fmt.Errorf("container %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of container %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckContainerExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccContainerID(s, resourceName)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetContainer(ctx, id); err != nil {
			return fmt.Errorf("container %d not found upstream: %w", id, err)
		}
		return nil
	}
}

// testAccCheckContainerDisappears deletes the container behind Terraform's
// back, which is what the next step's non-empty plan is asserting on.
func testAccCheckContainerDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccContainerID(s, resourceName)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		return c.DeleteContainer(ctx, id, nil)
	}
}
