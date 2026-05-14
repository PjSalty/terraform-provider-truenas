package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// updateTestFuncRE matches every TestAcc<Name>Resource_update or
// TestAcc<Name>_update function in the acc/resources test surface.
var updateTestFuncRE = regexp.MustCompile(`(?m)^func TestAcc\w+_update\(t \*testing\.T\)`)

// updateActionRE matches the plancheck assertion that the planned
// change is an in-place Update (not Replace, not no-op):
//
//	plancheck.ExpectResourceAction(<name>, plancheck.ResourceActionUpdate)
//
// We accept any whitespace between the comma and the Update constant.
var updateActionRE = regexp.MustCompile(`plancheck\.ExpectResourceAction\([^)]+plancheck\.ResourceActionUpdate\)`)

// updatePlanCheckMinimum is the floor for the count of _update acc tests
// that assert the change is an in-place Update via
// plancheck.ExpectResourceAction(... ResourceActionUpdate). A passing
// _update test that doesn't carry this check could be silently
// running a destroy+create cycle without the test author noticing —
// e.g., a Required-bumped-to-RequiresReplace bug slips in, and the
// end-state assertions still pass because the value is the same after
// recreate. ratcheting this up over time forces every _update test to
// prove the Update path, not just observe the end state.
//
// rolling out the assertion to a new _update test = bump this number
// by 1.
const updatePlanCheckMinimum = 36

// TestUpdatePlanCheckCoverage counts _update acc tests that assert
// ResourceActionUpdate via plancheck and fails if the count drops
// below updatePlanCheckMinimum.
func TestUpdatePlanCheckCoverage(t *testing.T) {
	var testFiles []string
	for _, dir := range []string{".", "../resources"} {
		matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		testFiles = append(testFiles, matches...)
	}

	var withCheck, without []string
	for _, f := range testFiles {
		base := filepath.Base(f)
		// Skip the invariant test files themselves.
		if strings.HasSuffix(base, "_invariant_test.go") ||
			strings.HasSuffix(base, "_coverage_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(src)
		if !updateTestFuncRE.MatchString(text) {
			continue
		}
		if updateActionRE.MatchString(text) {
			withCheck = append(withCheck, base)
		} else {
			without = append(without, base)
		}
	}
	sort.Strings(withCheck)
	sort.Strings(without)

	t.Logf("update-plan-shape coverage: %d _update tests with ResourceActionUpdate (floor %d)",
		len(withCheck), updatePlanCheckMinimum)
	for _, f := range withCheck {
		t.Logf("  OK       %s", f)
	}
	for _, f := range without {
		t.Logf("  MISSING  %s", f)
	}

	if len(withCheck) < updatePlanCheckMinimum {
		t.Fatalf("update-plan-shape coverage regressed: %d tests assert "+
			"ResourceActionUpdate, want at least %d. without the check, an "+
			"_update test can pass while silently running destroy+create "+
			"instead of in-place update — exactly the failure mode that "+
			"breaks the day someone accidentally adds RequiresReplace to a "+
			"Required attribute. add the assertion or, if you removed a "+
			"_update test, lower the floor with rationale.",
			len(withCheck), updatePlanCheckMinimum)
	}
}
