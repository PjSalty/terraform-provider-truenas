package resources_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// testAccAppCatalogApp returns a catalog app slug known to exist in the
// TRUENAS stable train. Override with TRUENAS_TEST_CATALOG_APP.
func testAccAppCatalogApp() string {
	if v := os.Getenv("TRUENAS_TEST_CATALOG_APP"); v != "" {
		return v
	}
	// syncthing: small, zero required values, long-lived in the stable
	// train. minio left the catalog sometime in 2026 and broke this test.
	return "syncthing"
}

func TestAccApp_basic(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_APPS") == "" {
		t.Skip("Skipping truenas_app acceptance test: set TRUENAS_TEST_APPS=1 and " +
			"ensure the target TrueNAS has the Apps service enabled with a pool configured.")
	}

	catalogApp := testAccAppCatalogApp()
	appName := "tf-acc-app"
	resourceName := "truenas_app.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAppDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccAppConfigBasic(appName, catalogApp),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "app_name", appName),
					resource.TestCheckResourceAttr(resourceName, "catalog_app", catalogApp),
					resource.TestCheckResourceAttr(resourceName, "train", "stable"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "state"),
				),
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"values", "catalog_app", "train", "remove_images", "remove_ix_volumes", "state"},
			},
		},
	})
}

// testAccAppWSClient builds a live wsclient from the same env vars the
// provider uses. Only for acceptance-test orchestration (docker
// gating, out-of-band compose edits), never for assertions the
// provider should make itself.
func testAccAppWSClient(t *testing.T) *wsclient.Client {
	t.Helper()
	url := os.Getenv("TRUENAS_URL")
	key := os.Getenv("TRUENAS_API_KEY")
	if url == "" || key == "" {
		t.Skip("TRUENAS_URL and TRUENAS_API_KEY must be set for acceptance tests")
	}
	insecure := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "true" ||
		os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "1"
	c, err := wsclient.New(context.Background(), url, key, insecure)
	if err != nil {
		t.Fatalf("connecting to TrueNAS for acc-test orchestration: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// testAccAppRequireDocker skips unless the target TrueNAS has the apps
// service configured on a pool (docker.config reports a pool). Custom
// compose apps need no catalog but still need docker placed on a pool.
func testAccAppRequireDocker(t *testing.T, c *wsclient.Client) {
	t.Helper()
	raw, err := c.Call(context.Background(), "docker.config", nil,
		wsclient.CallOptions{Read: true, Idempotent: true})
	if err != nil {
		t.Skipf("skipping: docker.config query failed, apps service likely unconfigured: %v", err)
	}
	var cfg struct {
		Pool string `json:"pool"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.Pool == "" {
		t.Skip("skipping: docker service has no pool configured on the target TrueNAS (docker.config pool is empty)")
	}
}

// testAccAppComposeYAML renders the minimal busybox compose used by
// the custom compose acceptance ladder. env varies between steps to
// prove compose content edits update in place.
func testAccAppComposeYAML(env string) string {
	return fmt.Sprintf("services:\n"+
		"  app:\n"+
		"    image: busybox:1.36\n"+
		"    command: [\"sleep\", \"infinity\"]\n"+
		"    restart: unless-stopped\n"+
		"    environment:\n"+
		"      DEMO_ENV: %q\n", env)
}

// testAccAppWaitRunning polls app.get_instance until the app settles
// RUNNING, tolerating the documented DEPLOYING-to-RUNNING transition.
// busybox sleep has no healthcheck, so it always settles. Bounded at
// 2 minutes.
func testAccAppWaitRunning(t *testing.T, c *wsclient.Client, appName string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		deadline := time.Now().Add(2 * time.Minute)
		last := ""
		for time.Now().Before(deadline) {
			app, err := c.GetApp(context.Background(), appName)
			if err != nil {
				return fmt.Errorf("polling app %q: %w", appName, err)
			}
			last = app.State
			if last == "RUNNING" {
				return nil
			}
			time.Sleep(5 * time.Second)
		}
		return fmt.Errorf("app %q never settled RUNNING, last state %q", appName, last)
	}
}

// TestAccApp_customCompose exercises the full custom compose ladder:
// create (settles RUNNING plus post-apply idempotency), in-place
// compose update (plancheck proves Update, not replace), import with
// the documented non-round-trip fields ignored, plan-time ExactlyOneOf
// rejections, semantic drift detection after an out-of-band
// app.update, and convergence back to the config.
//
// The reformat-only out-of-band case cannot be exercised live: the
// middleware stores the PARSED compose dict, so formatting does not
// exist server-side. Formatting tolerance is pinned at unit level
// (customtypes NormalizedYAML semantic equality, reconcile tests).
func TestAccApp_customCompose(t *testing.T) {
	if os.Getenv("TRUENAS_TEST_APPS") == "" {
		t.Skip("Skipping truenas_app acceptance test: set TRUENAS_TEST_APPS=1 and " +
			"ensure the target TrueNAS has the Apps service enabled with a pool configured.")
	}
	ws := testAccAppWSClient(t)
	testAccAppRequireDocker(t, ws)

	appName := "tf-acc-app-compose"
	resourceName := "truenas_app.custom"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAppDestroy(resourceName),
		Steps: []resource.TestStep{
			// basic: custom app settles RUNNING and is idempotent
			// after apply plus refresh
			{
				Config: testAccAppConfigCustom(appName, "one"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "app_name", appName),
					resource.TestMatchResourceAttr(resourceName, "state", regexp.MustCompile(`^(RUNNING|DEPLOYING)$`)),
					testAccAppWaitRunning(t, ws, appName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckNoResourceAttr(resourceName, "catalog_app"),
				),
			},
			// update: changing an environment variable in the compose
			// is an in-place update, never a replace
			{
				Config: testAccAppConfigCustom(appName, "two"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: testAccAppWaitRunning(t, ws, appName),
			},
			// import: custom_compose cannot round-trip, the middleware
			// stores the parsed compose, not the string
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"values", "catalog_app", "train",
					"remove_images", "remove_ix_volumes", "custom_compose", "state",
				},
			},
			// exactly-one-of rejections, plan-time only, nothing
			// reaches the API
			{
				Config:      testAccAppConfigCustomAndCatalog(appName),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
				PlanOnly:    true,
			},
			{
				Config:      testAccAppConfigNeither(appName),
				ExpectError: regexp.MustCompile(`Missing Attribute Configuration`),
				PlanOnly:    true,
			},
			// semantic drift: an out-of-band STRUCTURAL compose change
			// (different env value) must surface as a plan diff on the
			// next refresh
			{
				PreConfig: func() {
					_, err := ws.UpdateApp(context.Background(), appName, &truenas.AppUpdateRequest{
						CustomComposeConfigString: testAccAppComposeYAML("out-of-band"),
					})
					if err != nil {
						t.Fatalf("out-of-band app.update failed: %v", err)
					}
				},
				Config:             testAccAppConfigCustom(appName, "two"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// converge: applying the config repairs the drift in place
			// and the result is idempotent again
			{
				Config: testAccAppConfigCustom(appName, "two"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: testAccAppWaitRunning(t, ws, appName),
			},
		},
	})
}

func testAccAppConfigCustom(appName, env string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_app" "custom" {
  app_name       = %q
  custom_compose = %q
}
`, appName, testAccAppComposeYAML(env))
}

func testAccAppConfigCustomAndCatalog(appName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_app" "custom" {
  app_name       = %q
  catalog_app    = "minio"
  custom_compose = %q
}
`, appName, testAccAppComposeYAML("conflict"))
}

func testAccAppConfigNeither(appName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_app" "custom" {
  app_name = %q
}
`, appName)
}

func testAccCheckAppDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("app ID not set")
		}
		return nil
	}
}

func testAccAppConfigBasic(appName, catalogApp string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_app" "test" {
  app_name    = %q
  catalog_app = %q
  train       = "stable"
  version     = "latest"
  values      = "{}"
}
`, appName, catalogApp)
}
