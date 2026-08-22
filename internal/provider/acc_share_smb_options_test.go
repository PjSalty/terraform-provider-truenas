package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccShareSMBResource_options covers the purpose-specific options block.
//
// The plan-empty gate is the point of this test. Every options attribute is
// Optional+Computed, and TrueNAS fills the rest of the purpose's union member
// with its own defaults on create. If those come back into state as diffable
// changes, every subsequent plan shows a phantom diff on a share nobody
// touched, which is worse than the missing feature was.
func TestAccShareSMBResource_options(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestsmbopt")
	share := randomName("smboptshare")

	config := func(hosts string) string {
		return fmt.Sprintf(`
resource "truenas_dataset" "smb_opt" {
  pool       = "test"
  name       = %q
  share_type = "SMB"
}

resource "truenas_share_smb" "opt" {
  path    = truenas_dataset.smb_opt.mount_point
  name    = %q
  purpose = "DEFAULT_SHARE"
  enabled = true

  options = {
    hostsallow         = %s
    aapl_name_mangling = true
  }
}
`, ds, share, hosts)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`["10.0.0.0/8"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_smb.opt", "options.hostsallow.#", "1"),
					resource.TestCheckResourceAttr("truenas_share_smb.opt", "options.hostsallow.0", "10.0.0.0/8"),
					resource.TestCheckResourceAttr("truenas_share_smb.opt", "options.aapl_name_mangling", "true"),
					// Not set here, so the server default must be readable
					// rather than unknown or missing.
					resource.TestCheckResourceAttrSet("truenas_share_smb.opt", "options.hostsdeny.#"),
				),
			},
			{
				// Changing one option must be an in-place update, not a
				// replacement, and must settle to an empty plan again.
				Config: config(`["10.0.0.0/8", "192.168.0.0/16"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_share_smb.opt", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_share_smb.opt", "options.hostsallow.#", "2"),
			},
			{
				ResourceName:      "truenas_share_smb.opt",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccShareSMBResource_external is the case that could not be expressed at
// all before the options block: EXTERNAL_SHARE redirects clients to another
// server, so it needs options.remote_path and the literal path "EXTERNAL".
func TestAccShareSMBResource_external(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	share := randomName("smbext")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_share_smb" "ext" {
  path    = "EXTERNAL"
  name    = %q
  purpose = "EXTERNAL_SHARE"

  options = {
    remote_path = ["fileserver.example.com\\archive"]
  }
}
`, share),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_smb.ext", "purpose", "EXTERNAL_SHARE"),
					resource.TestCheckResourceAttr("truenas_share_smb.ext", "options.remote_path.#", "1"),
					resource.TestCheckResourceAttr("truenas_share_smb.ext", "options.remote_path.0", `fileserver.example.com\archive`),
				),
			},
			{
				ResourceName:      "truenas_share_smb.ext",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccShareSMBResource_timemachine exercises a purpose whose options are a
// different union member, so a field accepted here would be rejected on the
// DEFAULT_SHARE above.
//
// TIMEMACHINE_SHARE also needs aapl_extensions on the global SMB config, and
// without it the create fails with an EINVAL that names `purpose` rather than
// the setting it actually wants. The dependency is declared here for the same
// reason it is documented: it is not discoverable from the share alone.
func TestAccShareSMBResource_timemachine(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	ds := randomName("acctestsmbtm")
	share := randomName("smbtm")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_smb_config" "aapl" {
  aapl_extensions = true
}

resource "truenas_dataset" "smb_tm" {
  pool       = "test"
  name       = %q
  share_type = "SMB"
}

resource "truenas_share_smb" "tm" {
  depends_on = [truenas_smb_config.aapl]

  path    = truenas_dataset.smb_tm.mount_point
  name    = %q
  purpose = "TIMEMACHINE_SHARE"

  options = {
    auto_snapshot         = true
    auto_dataset_creation = true
    dataset_naming_schema = "%%D/%%U"
  }
}
`, ds, share),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_share_smb.tm", "options.auto_snapshot", "true"),
					resource.TestCheckResourceAttr("truenas_share_smb.tm", "options.auto_dataset_creation", "true"),
					resource.TestCheckResourceAttr("truenas_share_smb.tm", "options.dataset_naming_schema", "%D/%U"),
					// vuid belongs to this purpose and is server-assigned.
					resource.TestCheckResourceAttrSet("truenas_share_smb.tm", "options.vuid"),
				),
			},
		},
	})
}
