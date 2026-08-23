package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestAccSMBShare_basic(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smb-basic"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccSMBShareConfigBasic(pool, datasetName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", "tf-acc-smb-basic"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "browsable", "true"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
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

func TestAccSMBShare_update(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smb-update"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create with readonly false
			{
				Config: testAccSMBShareConfigReadOnly(pool, datasetName, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
					resource.TestCheckResourceAttr(resourceName, "comment", "acc test share"),
				),
			},
			// Update: set readonly true
			{
				Config: testAccSMBShareConfigReadOnly(pool, datasetName, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "readonly", "true"),
				),
			},
		},
	})
}

func testAccCheckSMBShareDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("SMB share ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("SMB share ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetSMBShare(ctx, id)
		if err == nil {
			return fmt.Errorf("SMB share %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of SMB share %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckSMBShareExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetSMBShare(ctx, id); err != nil {
			return fmt.Errorf("SMB share %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckSMBShareDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteSMBShare(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of SMB share %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccSMBShare_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-smbshare-disappears"
	resourceName := "truenas_share_smb.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccSMBShareConfigBasic(pool, datasetName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: testAccCheckSMBShareExists(resourceName),
			},
			{
				Config:             testAccSMBShareConfigBasic(pool, datasetName),
				Check:              testAccCheckSMBShareDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccSMBShareConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "smb_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_smb" "test" {
  path = truenas_dataset.smb_parent.mount_point
  name = %q
}
`, pool, datasetName, datasetName)
}

func testAccSMBShareConfigReadOnly(pool, datasetName string, readOnly bool) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "smb_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_smb" "test" {
  path     = truenas_dataset.smb_parent.mount_point
  name     = %q
  readonly = %t
  comment  = "acc test share"
}
`, pool, datasetName, datasetName, readOnly)
}

// TestAccSMBShare_purposePreset_25_10 cycles through every SMB share
// purpose preset that TrueNAS SCALE 25.10 declares via
// `sharing.smb.presets`. A single regression in either the upstream
// preset registry OR our stringvalidator.OneOf list surfaces here
// as the specific preset that broke.
//
// History: SCALE 25.10 introduced a complete overhaul of the SMB
// preset vocabulary. The earlier list (ENHANCED_TIMEMACHINE,
// LEGACY_SMB_WHITELIST, MULTI_PROTOCOL_NFS, MULTI_PROTOCOL_AFP,
// PRIVATE_DATASETS, NO_PRESET, TIMEMACHINE) has ZERO overlap with
// the 25.10 list. Our provider's validator was caught with the old
// vocabulary; this test exercises the new vocabulary to prevent
// silent regression on future SCALE upgrades.
func TestAccSMBShare_purposePreset_25_10(t *testing.T) {
	pool := testAccDatasetPool()
	// 25.10 preset registry verified via `sharing.smb.presets`
	// against a live 25.10.0 instance on 2026-06-08. Per-preset
	// requirements documented inline; the first acc run surfaced:
	//   - TIMEMACHINE_SHARE: requires `aapl_extensions: true` on the
	//     global `smb.config` (Apple SMB2/3 extensions). The provider
	//     DOES manage that, on truenas_smb_config, so the test composes
	//     one and depends on it rather than skipping.
	//
	//     This comment previously said the provider could not manage the
	//     toggle, and that the preset was gated behind
	//     TRUENAS_TEST_SMB_AAPL=1. Both halves were false: smb_config
	//     exposes aapl_extensions, and no such env gate was ever
	//     implemented, so setting that variable did nothing at all.
	//   - EXTERNAL_SHARE: needs options.remote_path and the literal path
	//     "EXTERNAL". That used to be skipped as "the provider exposes no
	//     preset-options map", which was true when it was written and is not
	//     now: truenas_share_smb has an options block. A test that claims to
	//     cycle EVERY preset while skipping the one that was broken is worse
	//     than one that admits the gap, so it runs.
	//   - VEEAM_REPOSITORY_SHARE: requires a TrueNAS enterprise
	//     license. Always skipped on community edition.
	type presetCase struct {
		name      string
		skip      string // non-empty => skip with reason
		needsAAPL bool   // preset needs aapl_extensions on the global SMB config
		external  bool   // proxies to a remote server rather than local storage
	}
	presets := []presetCase{
		{name: "DEFAULT_SHARE"},
		{name: "LEGACY_SHARE"},
		{name: "TIMEMACHINE_SHARE", needsAAPL: true},
		{name: "MULTIPROTOCOL_SHARE"},
		{name: "PRIVATE_DATASETS_SHARE"},
		{name: "EXTERNAL_SHARE", external: true},
		{name: "TIME_LOCKED_SHARE"},
		{name: "VEEAM_REPOSITORY_SHARE", skip: "requires TrueNAS enterprise license"},
		// Added upstream in 25.10.1, so a 25.10.0 box rejects it.
		{name: "FCP_SHARE", needsAAPL: true},
	}
	for i, pc := range presets {
		preset := pc.name
		t.Run(preset, func(t *testing.T) {
			if pc.skip != "" {
				t.Skip(pc.skip)
			}
			// Each preset gets its own unique dataset so the shares
			// don't collide. Use the preset index to keep names short
			// (some presets like VEEAM_REPOSITORY_SHARE would push the
			// dataset name above SMB share-name limits if we
			// embedded the full preset).
			ds := fmt.Sprintf("tf-acc-smb-preset-%d", i)
			resourceName := "truenas_share_smb.test"

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckSMBShareDestroy(resourceName),
				Steps: []resource.TestStep{
					{
						Config: testAccSMBShareConfigPreset(pool, ds, preset, pc.needsAAPL, pc.external),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet(resourceName, "id"),
							resource.TestCheckResourceAttr(resourceName, "purpose", preset),
						),
					},
				},
			})
		})
	}
}

func testAccSMBShareConfigPreset(pool, datasetName, preset string, needsAAPL, external bool) string {
	// An EXTERNAL_SHARE is a DFS redirect: there is no local dataset to make,
	// the path is the literal sentinel, and remote_path carries the targets.
	if external {
		return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_share_smb" "test" {
  path    = "EXTERNAL"
  name    = %q
  purpose = %q
  comment = "preset compat test"

  options = {
    remote_path = ["fileserver.example.com\\archive"]
  }
}
`, datasetName, preset)
	}

	// TIMEMACHINE_SHARE is rejected outright unless Apple SMB2/3
	// extensions are enabled globally. smb_config is a singleton, so this
	// flips a box-wide setting for the duration of the test; the
	// framework resets it on destroy.
	aapl, dependsOn := "", ""
	if needsAAPL {
		aapl = `
resource "truenas_smb_config" "aapl" {
  aapl_extensions = true
}
`
		dependsOn = "\n  depends_on = [truenas_smb_config.aapl]"
	}
	return fmt.Sprintf(`
provider "truenas" {}
%s

resource "truenas_dataset" "smb_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_smb" "test" {
  path    = truenas_dataset.smb_parent.mount_point
  name    = %q
  purpose = %q
  comment = "preset compat test"%s
}
`, aapl, pool, datasetName, datasetName, preset, dependsOn)
}
