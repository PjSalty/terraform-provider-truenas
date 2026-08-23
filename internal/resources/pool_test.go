package resources_test

import (
	"testing"
)

// TestAccPool_basic is a placeholder. Writing it needs dedicated spare disks
// on the target TrueNAS, which is why it does not exist yet.
//
// It does NOT read TRUENAS_TEST_POOL_DISKS. Nothing in the tree does. An
// earlier version of this comment told operators to set it, which could not
// have worked: the function has no body, so there is nothing for an
// environment variable to enable.
func TestAccPool_basic(t *testing.T) {
	t.Skip("no acceptance test yet: needs dedicated test disks on the target box. There is no body here to enable, so setting TF_ACC changes nothing; writing the test is the work.")
}
