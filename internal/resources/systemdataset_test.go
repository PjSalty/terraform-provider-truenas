package resources_test

import (
	"testing"
)

// TestAccSystemDataset_basic is a placeholder. Moving the system dataset
// between pools is a system-wide change that can disrupt services, so it would
// only ever run against a dedicated test VM.
//
// It does NOT read TRUENAS_TEST_SYSTEMDATASET_POOL. Nothing in the tree does.
// An earlier version of this comment told operators to set it, which could not
// have worked: the function has no body.
func TestAccSystemDataset_basic(t *testing.T) {
	t.Skip("no acceptance test yet: mutating the system dataset changes global config on a shared box. There is no body here to enable, so setting TF_ACC changes nothing; writing the test is the work.")
}
