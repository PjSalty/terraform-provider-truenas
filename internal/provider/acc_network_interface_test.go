package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNetworkInterfaceResource_basic is an IMPORT-ONLY test against a
// pre-existing interface on the TrueNAS SCALE test VM. The
// network_interface resource is marked Beta and its create/update/delete
// cycle is not exercised against the live VM because modifying the
// active management interface risks cutting off the provider's own API
// access mid-test (the TrueNAS commit/checkin workflow is designed for
// interactive console use, not automated rollback).
//
// The interface name is taken from TRUENAS_TEST_IFACE; when that env
// var is unset the test is skipped so local developer runs do not hard
// fail. In CI, set TRUENAS_TEST_IFACE=ens3 (or whatever the primary
// NIC is on the test VM).
func TestAccNetworkInterfaceResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	iface := os.Getenv("TRUENAS_TEST_IFACE")
	if iface == "" {
		t.Skip("TRUENAS_TEST_IFACE not set — skipping network_interface import test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Config must declare a minimal plan so the import can
				// match. type is the only required attribute; we use
				// PHYSICAL for the management NIC.
				Config: fmt.Sprintf(`
resource "truenas_network_interface" "test" {
  name = %q
  type = "PHYSICAL"
}
`, iface),
				ResourceName:      "truenas_network_interface.test",
				ImportState:       true,
				ImportStateId:     iface,
				ImportStateVerify: false,
			},
		},
	})
}

// TestAccNetworkInterfaceResource_update verifies that the description
// field on a pre-existing interface can be toggled in place without
// affecting IP configuration or triggering a commit/checkin rollback.
// The description is the only field safe to mutate on a shared test VM
// since every other field risks cutting SSH/API access.
//
// A third step restores the original description ("") so the shared
// test VM is left in a predictable state.
//
// Skipped unless TRUENAS_TEST_IFACE is set — same rationale as _basic.
// No _disappears test: deleting an interface mid-test would very
// likely break the provider's own connection to the TrueNAS API.
func TestAccNetworkInterfaceResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	iface := os.Getenv("TRUENAS_TEST_IFACE")
	if iface == "" {
		t.Skip("TRUENAS_TEST_IFACE not set — skipping network_interface update test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_network_interface" "test" {
  name        = %q
  type        = "PHYSICAL"
  description = "acctest-initial"
}
`, iface),
				Check: resource.TestCheckResourceAttr("truenas_network_interface.test", "description", "acctest-initial"),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_network_interface" "test" {
  name        = %q
  type        = "PHYSICAL"
  description = "acctest-updated"
}
`, iface),
				Check: resource.TestCheckResourceAttr("truenas_network_interface.test", "description", "acctest-updated"),
			},
			{
				// Restore empty description so the shared test VM is
				// left in a predictable state for subsequent runs.
				Config: fmt.Sprintf(`
resource "truenas_network_interface" "test" {
  name        = %q
  type        = "PHYSICAL"
  description = ""
}
`, iface),
				Check: resource.TestCheckResourceAttr("truenas_network_interface.test", "description", ""),
			},
		},
	})
}
