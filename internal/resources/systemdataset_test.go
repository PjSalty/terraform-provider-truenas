package resources_test

import (
	"testing"
)

// TestAccSystemDataset_basic verifies reading the current system dataset
// config. This is skipped by default because moving the system dataset
// between pools is a system-wide change that may disrupt services; it
// should only be run against a dedicated test TrueNAS VM. Enable with
// TF_ACC=1 and TRUENAS_TEST_SYSTEMDATASET_POOL set.
func TestAccSystemDataset_basic(t *testing.T) {
	t.Skip("no acceptance test yet: mutating the system dataset changes global config on a shared box. There is no body here to enable, so setting TF_ACC changes nothing; writing the test is the work.")
}
