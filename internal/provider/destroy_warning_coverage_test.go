package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// destroyWarnFloor is the minimum number of resource files that MUST
// call planhelpers.WarnOnDestroy from their ModifyPlan hook. This is
// a SLO-style ratchet — raising it is cheap (commit the new floor
// with the new WarnOnDestroy call), lowering it requires a documented
// reason in the PR comment. The same mechanism as
// TestIdempotencyCheckCoverage and TestConfigValidatorsCoverage.
//
// Destructive resources (dataset, zvol, pool, vm, share_*, user,
// group, iscsi_*, replication, cloud_*, cronjob, scrub_task,
// snapshot_task, init_script) MUST all be in the set. Adding
// WarnOnDestroy to a new destructive resource = bump this number by 1.
const destroyWarnFloor = 22

// TestDestroyWarningCoverage counts resource files that call
// planhelpers.WarnOnDestroy and fails if the count drops below
// destroyWarnFloor. The plan-time destroy warning is the
// complementary rail to client-layer DestroyProtection: the
// protection BLOCKS the wire call (post-plan, at apply time), while
// the warning SHOWS the operator what would be destroyed before
// they ever type `terraform apply`. Losing coverage on either rail
// reintroduces silent destruction risk.
func TestDestroyWarningCoverage(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	var files []string
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		if strings.Contains(string(src), "planhelpers.WarnOnDestroy(") {
			files = append(files, base)
		}
	}
	if len(files) < destroyWarnFloor {
		t.Fatalf("plan-time destroy warning coverage dropped: have %d, "+
			"want ≥ %d. Files with WarnOnDestroy: %v\n\n"+
			"This is a ratchet. Either add a WarnOnDestroy call to at "+
			"least one more destructive resource (preferred — bump the "+
			"floor) or, if you intentionally removed one, lower "+
			"destroyWarnFloor in this file with a PR-comment justification.",
			len(files), destroyWarnFloor, files)
	}
	t.Logf("WarnOnDestroy coverage: %d resource files (floor %d)", len(files), destroyWarnFloor)
}
