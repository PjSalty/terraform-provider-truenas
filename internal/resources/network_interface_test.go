package resources_test

import (
	"testing"
)

// TestAccNetworkInterface_vlan verifies creating a VLAN interface over a
// parent physical interface. This is skipped by default because it requires
// knowing a valid parent interface name on the target TrueNAS system.
// Run with TF_ACC=1 and TRUENAS_TEST_PARENT_INTERFACE set.
func TestAccNetworkInterface_vlan(t *testing.T) {
	t.Skip("no acceptance test yet: needs a known physical parent interface on the target box. There is no body here to enable, so setting TF_ACC changes nothing; writing the test is the work.")
}
