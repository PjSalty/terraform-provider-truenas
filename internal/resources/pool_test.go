package resources_test

import (
	"testing"
)

// TestAccPool_basic verifies end-to-end pool creation with a mirror-of-2
// topology. This is skipped by default because it requires dedicated test
// disks on the target TrueNAS SCALE system which may not be available in
// every CI environment. Run with TF_ACC=1 and TRUENAS_TEST_POOL_DISKS set
// to a comma-separated list of disk device names (e.g. "sdb,sdc").
func TestAccPool_basic(t *testing.T) {
	t.Skip("no acceptance test yet: needs dedicated test disks on the target box. There is no body here to enable, so setting TF_ACC changes nothing; writing the test is the work.")
}
