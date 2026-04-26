package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccVMResource_basic creates a minimal VM with no devices (devices
// are managed independently via truenas_vm_device). VM names are
// constrained to alphanumeric characters.
func TestAccVMResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctvm%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_vm" "test" {
  name   = %q
  vcpus  = 1
  cores  = 1
  memory = 128
}
`, name),
				// Structured plan assertions: the VM must plan as a
				// Create action, and the computed `id` attribute must be
				// unknown before apply (since it is assigned by the API).
				// The `name` attribute must resolve to our generated
				// value at plan time.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_vm.test", plancheck.ResourceActionCreate),
						plancheck.ExpectUnknownValue("truenas_vm.test", tfjsonpath.New("id")),
						plancheck.ExpectKnownValue(
							"truenas_vm.test",
							tfjsonpath.New("name"),
							knownvalue.StringExact(name),
						),
						plancheck.ExpectKnownValue(
							"truenas_vm.test",
							tfjsonpath.New("memory"),
							knownvalue.Int64Exact(128),
						),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_vm.test", "name", name),
					resource.TestCheckResourceAttr("truenas_vm.test", "vcpus", "1"),
					resource.TestCheckResourceAttr("truenas_vm.test", "cores", "1"),
					resource.TestCheckResourceAttr("truenas_vm.test", "memory", "128"),
					resource.TestCheckResourceAttrSet("truenas_vm.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_vm.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccVMResource_update exercises description + memory round-trip to
// verify the VM PUT path is wired for updatable fields without causing a
// replacement.
func TestAccVMResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctvm%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_vm" "test" {
  name        = %q
  description = "initial"
  vcpus       = 1
  cores       = 1
  memory      = 128
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_vm.test", "description", "initial"),
					resource.TestCheckResourceAttr("truenas_vm.test", "memory", "128"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_vm" "test" {
  name        = %q
  description = "updated"
  vcpus       = 1
  cores       = 1
  memory      = 256
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_vm.test", "description", "updated"),
					resource.TestCheckResourceAttr("truenas_vm.test", "memory", "256"),
				),
			},
		},
	})
}

// TestAccVMResource_disappears deletes the VM out-of-band and verifies
// the provider detects drift via a non-empty follow-up plan.
func TestAccVMResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctvm%d", shortSuffix())
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_vm" "test" {
  name   = %q
  vcpus  = 1
  cores  = 1
  memory = 128
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_vm.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_vm.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parsing id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteVM(ctx, id, nil)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
