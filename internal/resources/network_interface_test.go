package resources_test

import (
	"testing"
)

// TestAccNetworkInterface_vlan is a placeholder. Writing it needs a known
// physical parent interface on the target box.
//
// Setting TRUENAS_TEST_PARENT_INTERFACE does not enable this one. That
// variable is real, but it gates the data source test in
// network_interface_data_test.go; this function has no body to enable.
func TestAccNetworkInterface_vlan(t *testing.T) {
	t.Skip("no acceptance test yet: needs a known physical parent interface on the target box. There is no body here to enable, so setting TF_ACC changes nothing; writing the test is the work.")
}
