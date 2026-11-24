package resources_test

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// A disk is hardware, so these tests adopt one of the disks the test box
// already has instead of creating one, and Delete is a no-op. Whatever the
// last applied step sets is therefore what the box keeps: every test ends on
// a step that writes back the values the disk had when the test began.
//
// The test box has virtual disks. disk.update stores the settings and, when
// the standby timer or the power level changed, runs hdparm against the
// device in the background with its exit status ignored. A virtual disk that
// does not understand hdparm is therefore not an error, and the settings
// still read back. The tests change the description and the standby timer,
// which has no effect on the box's network, API or boot.

// testAccDisk looks a disk up at run time, because its identifier is not
// known until the box exists. It takes the first disk by device name.
func testAccDisk(t *testing.T) *truenas.Disk {
	t.Helper()
	// resource.Test skips without TF_ACC, but this runs before it.
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests skipped unless TF_ACC is set")
	}
	testAccPreCheck(t)

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	disks, err := c.ListDisks(ctx)
	if err != nil {
		t.Fatalf("listing disks: %v", err)
	}
	if len(disks) == 0 {
		t.Skip("TrueNAS reports no disks, skipping")
	}
	sort.Slice(disks, func(i, j int) bool { return disks[i].Name < disks[j].Name })

	// disk.query leaves the pool out; the by-identifier read fills it in.
	disk, err := c.GetDisk(ctx, disks[0].Identifier)
	if err != nil {
		t.Fatalf("reading disk %q: %v", disks[0].Identifier, err)
	}
	return disk
}

// hclString quotes s for a Terraform string literal. %q is not enough: it
// leaves "${" and "%{" alone, and HCL would read them as interpolation.
func hclString(s string) string {
	q := fmt.Sprintf("%q", s)
	q = strings.ReplaceAll(q, "${", "$${")
	return strings.ReplaceAll(q, "%{", "%%{")
}

// testAccDiskConfig renders the resource. An empty description or standby
// leaves the attribute out, so the test also covers "unset keeps the
// current value".
func testAccDiskConfig(identifier, description, standby string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nprovider \"truenas\" {}\n\nresource \"truenas_disk\" \"test\" {\n  identifier  = %s\n", hclString(identifier))
	if description != "" {
		fmt.Fprintf(&b, "  description = %s\n", hclString(description))
	}
	if standby != "" {
		fmt.Fprintf(&b, "  hddstandby  = %q\n", standby)
	}
	b.WriteString("}\n")
	return b.String()
}

// testAccDiskRestore renders the disk as the test found it, every setting
// explicit, so the final step puts back what the test changed.
func testAccDiskRestore(d *truenas.Disk) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_disk" "test" {
  identifier   = %s
  description  = %s
  hddstandby   = %q
  advpowermgmt = %q
}
`, hclString(d.Identifier), hclString(d.Description), d.HDDStandby, d.AdvPowerMgmt)
}

// testAccOtherStandby is a standby timer different from the disk's own.
func testAccOtherStandby(current string) string {
	if current == "60" {
		return "120"
	}
	return "60"
}

func TestAccDisk_basic(t *testing.T) {
	disk := testAccDisk(t)
	resourceName := "truenas_disk.test"
	standby := testAccOtherStandby(disk.HDDStandby)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDiskConfig(disk.Identifier, "terraform acceptance test", standby),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", disk.Identifier),
					resource.TestCheckResourceAttr(resourceName, "identifier", disk.Identifier),
					resource.TestCheckResourceAttr(resourceName, "description", "terraform acceptance test"),
					resource.TestCheckResourceAttr(resourceName, "hddstandby", standby),
					// Not configured, so it keeps what the disk has.
					resource.TestCheckResourceAttr(resourceName, "advpowermgmt", disk.AdvPowerMgmt),
					resource.TestCheckResourceAttr(resourceName, "serial", disk.Serial),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     disk.Identifier,
				ImportStateVerify: true,
			},
			{
				Config: testAccDiskRestore(disk),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", disk.Description),
					resource.TestCheckResourceAttr(resourceName, "hddstandby", disk.HDDStandby),
					resource.TestCheckResourceAttr(resourceName, "advpowermgmt", disk.AdvPowerMgmt),
				),
			},
		},
	})
}

// Both settings change in place, and the disk is not replaced.
func TestAccDisk_update(t *testing.T) {
	disk := testAccDisk(t)
	resourceName := "truenas_disk.test"
	standby := testAccOtherStandby(disk.HDDStandby)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDiskRestore(disk),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: testAccDiskConfig(disk.Identifier, "terraform acceptance update", standby),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "terraform acceptance update"),
					resource.TestCheckResourceAttr(resourceName, "hddstandby", standby),
				),
			},
			{
				Config: testAccDiskRestore(disk),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", disk.Description),
					resource.TestCheckResourceAttr(resourceName, "hddstandby", disk.HDDStandby),
				),
			},
		},
	})
}

// A configuration that sets only the identifier adopts the disk and leaves
// every setting as it is, and still plans nothing afterwards.
func TestAccDisk_adoptKeepsSettings(t *testing.T) {
	disk := testAccDisk(t)
	resourceName := "truenas_disk.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDiskConfig(disk.Identifier, "", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", disk.Description),
					resource.TestCheckResourceAttr(resourceName, "hddstandby", disk.HDDStandby),
					resource.TestCheckResourceAttr(resourceName, "advpowermgmt", disk.AdvPowerMgmt),
				),
			},
		},
	})
}

// A disk cannot be created. An identifier no disk has fails the apply with
// a message that says so, before anything is written.
func TestAccDisk_unknownIdentifierRejected(t *testing.T) {
	testAccDisk(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDiskConfig("{serial}no-such-disk-terraform-acc", "x", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ExpectError: regexp.MustCompile(`TrueNAS has no disk with identifier`),
			},
		},
	})
}

// The enumerations fail at plan time, before anything is written. TrueNAS
// reads them back upper-cased, so another spelling would never settle.
func TestAccDisk_invalidSettingsRejected(t *testing.T) {
	disk := testAccDisk(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDiskConfig(disk.Identifier, "", "15"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ExpectError: regexp.MustCompile(`hddstandby`),
				PlanOnly:    true,
			},
			{
				Config: testAccDiskConfig(disk.Identifier, "", "Always On"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ExpectError: regexp.MustCompile(`hddstandby`),
				PlanOnly:    true,
			},
		},
	})
}
