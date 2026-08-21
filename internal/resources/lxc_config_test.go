package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
)

// LXC containers are new in TrueNAS 26.0. On 25.10 and older the lxc
// namespace does not exist, so these skip rather than fail: a red suite on
// a 25.10 box would say nothing about the provider.
func testAccLXCConfigPreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	if _, err := c.GetLXCConfig(ctx); err != nil {
		t.Skipf("LXC unavailable on this server, skipping: %v", err)
	}
}

// preferred_pool is the only field safe to exercise here. Writing bridge,
// v4_network or v6_network would reconfigure container networking on the
// test box for every later run; preferred_pool only decides where new
// container storage lands, so setting it to the acceptance pool is both
// meaningful and reversible.
func TestAccLXCConfig_basic(t *testing.T) {
	pool := testAccDatasetPool()
	resourceName := "truenas_lxc_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccLXCConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLXCConfig(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "lxc_config"),
					resource.TestCheckResourceAttr(resourceName, "preferred_pool", pool),
					// bridge is nullable upstream and empty on a box with no
					// bridge interface. It must still come back KNOWN, or
					// every later plan shows it as "(known after apply)".
					// TestCheckResourceAttrSet would reject the empty value
					// here, so this asserts the value itself.
					resource.TestCheckResourceAttr(resourceName, "bridge", ""),
					resource.TestCheckResourceAttrSet(resourceName, "v4_network"),
					resource.TestCheckResourceAttrSet(resourceName, "v6_network"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "lxc_config",
				ImportStateVerify: true,
			},
		},
	})
}

// A second apply with the same config must be a no-op. This is the check
// that catches a refresh writing a field back in a shape the server did
// not return, which shows up as a permanent diff rather than an error.
func TestAccLXCConfig_idempotent(t *testing.T) {
	pool := testAccDatasetPool()
	resourceName := "truenas_lxc_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccLXCConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLXCConfig(pool),
				Check:  resource.TestCheckResourceAttr(resourceName, "preferred_pool", pool),
			},
			{
				Config:             testAccLXCConfig(pool),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccLXCConfig(pool string) string {
	return testAccLXCConfigWithNetwork(pool, lxcDefaultV4Network)
}

func testAccLXCConfigWithNetwork(pool, v4 string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_lxc_config" "test" {
  preferred_pool = %q
  v4_network     = %q
}
`, pool, v4)
}

// The network TrueNAS ships as the default for the managed container
// bridge. Every test that changes v4_network restores this as its last
// applied step, because Delete on a singleton is a no-op: whatever the
// final step sets is what the box keeps.
const lxcDefaultV4Network = "172.200.0.0/24"

// Picked to stay clear of the homelab ranges the test VM itself sits on,
// so TrueNAS's own overlap check against system IPs does not reject it.
const lxcAltV4Network = "172.201.0.0/24"

// preferred_pool cannot be the update lever here: TrueNAS boxes commonly
// have a single data pool, so there is no second value to toggle to.
// v4_network is the field with two safe values, and the second step puts
// the default back.
func TestAccLXCConfig_update(t *testing.T) {
	pool := testAccDatasetPool()
	resourceName := "truenas_lxc_config.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccLXCConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLXCConfigWithNetwork(pool, lxcAltV4Network),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "v4_network", lxcAltV4Network),
					resource.TestCheckResourceAttr(resourceName, "preferred_pool", pool),
				),
			},
			{
				Config: testAccLXCConfigWithNetwork(pool, lxcDefaultV4Network),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "v4_network", lxcDefaultV4Network),
					resource.TestCheckResourceAttr(resourceName, "preferred_pool", pool),
				),
			},
		},
	})
}

// lxc.bridge_choices advertises "[AUTO]" for the managed bridge, but
// TrueNAS stores it as null and reads it back as "", so a config using it
// can never converge. The provider cannot rewrite it either: the framework
// rejects a planned value that differs from a non-null config value. So it
// must fail at validate time with a message naming the spelling that
// works, and this pins that it fails BEFORE any apply touches the server.
func TestAccLXCConfig_bridgeAutoSentinelRejected(t *testing.T) {
	pool := testAccDatasetPool()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccLXCConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "truenas" {}

resource "truenas_lxc_config" "test" {
  preferred_pool = %q
  bridge         = "[AUTO]"
}
`, pool),
				ExpectError: regexp.MustCompile(`bridge = ""`),
				PlanOnly:    true,
			},
		},
	})
}

// The data source must observe what the resource wrote. Reading through a
// second code path is what proves the resource is not just round-tripping
// its own plan.
func TestAccLXCConfigDataSource_matchesResource(t *testing.T) {
	pool := testAccDatasetPool()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccLXCConfigPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLXCConfig(pool) + `
data "truenas_lxc_config" "test" {
  depends_on = [truenas_lxc_config.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_lxc_config.test", "id", "lxc_config"),
					resource.TestCheckResourceAttrPair(
						"data.truenas_lxc_config.test", "preferred_pool",
						"truenas_lxc_config.test", "preferred_pool",
					),
				),
			},
		},
	})
}
