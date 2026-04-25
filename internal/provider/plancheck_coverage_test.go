package provider

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestIdempotencyCheckCoverage reports how many acc_*_test.go files assert
// the apply-idempotency invariant via
//
//	ConfigPlanChecks{
//	    PostApplyPostRefresh: []plancheck.PlanCheck{
//	        plancheck.ExpectEmptyPlan(),
//	    },
//	}
//
// The check catches Read/Create/Update shape mismatches, server defaults
// that leak as phantom diffs, computed fields that round-trip
// differently, and the whole "terraform plan is never clean" family of
// provider bugs.
//
// This test is a RATCHET, not a blind gate. Not every acceptance test
// has been validated against a real TrueNAS instance yet, so rollout is
// incremental. Each commit that adds ExpectEmptyPlan to a new acc test
// MUST also bump the idempotencyCheckMinimum constant below so the
// coverage floor only ever goes up.
//
// To see which tests are missing the check, run:
//
//	go test -v -run TestIdempotencyCheckCoverage ./internal/provider/
//
// and grep for "MISSING" in the output.

// idempotencyCheckMinimum is the floor for the number of acc_*_test.go
// files that MUST contain an apply-idempotency check. Increase this
// number whenever you roll the check out to another acc test — never
// decrease it, that would regress battle-hardening coverage for no
// reason. The starting floor matches the three pattern-proof
// resources shipped in b6938a9 (dataset, user, share_smb).
const idempotencyCheckMinimum = 3

func TestIdempotencyCheckCoverage(t *testing.T) {
	matches, err := filepath.Glob("acc_*_test.go")
	if err != nil {
		t.Fatalf("glob acc tests: %v", err)
	}
	// acc_helpers_test.go is infrastructure (pre/skip/rand), not a
	// resource-level test — skip it.
	var files []string
	for _, m := range matches {
		if strings.HasSuffix(m, "acc_helpers_test.go") {
			continue
		}
		files = append(files, m)
	}

	var withCheck, without []string
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if bytes.Contains(src, []byte("PostApplyPostRefresh")) &&
			bytes.Contains(src, []byte("plancheck.ExpectEmptyPlan")) {
			withCheck = append(withCheck, f)
		} else {
			without = append(without, f)
		}
	}
	sort.Strings(withCheck)
	sort.Strings(without)

	t.Logf("apply-idempotency coverage: %d/%d acc tests (%0.1f%%)",
		len(withCheck), len(files), 100.0*float64(len(withCheck))/float64(len(files)))
	for _, f := range withCheck {
		t.Logf("  OK       %s", f)
	}
	for _, f := range without {
		t.Logf("  MISSING  %s", f)
	}

	if len(withCheck) < idempotencyCheckMinimum {
		t.Fatalf("idempotency coverage regressed: %d acc tests have ExpectEmptyPlan, "+
			"want at least %d (idempotencyCheckMinimum). Adding a new acc test without "+
			"the check is fine — just do not REMOVE one.",
			len(withCheck), idempotencyCheckMinimum)
	}
}
