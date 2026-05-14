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

// updatePlanCheckExclusions lists every acc_*_test.go file whose
// _update test does NOT carry plancheck.ResourceActionUpdate, with
// rationale. Every other acc test file with an _update function
// MUST assert ResourceActionUpdate on the change step.
//
// Legitimate exclusion categories:
//   - The "update" step is a no-op by design (same-value re-apply
//     because the only safe mutation against a shared test VM is none).
//   - The change attribute is RequiresReplace, so the plan is Replace,
//     not Update. (These could instead assert ResourceActionReplace —
//     left as a follow-up.)
//   - The resource is a data source / read-only / gated test that
//     can't be safely mutated against the test VM.
var updatePlanCheckExclusions = map[string]string{
	"acc_kmip_config_test.go":        "_update is t.Skip stub: SCALE 25.10 KMIP does not persist user-set fields while enabled=false; see test docstring",
	"acc_network_interface_test.go":  "modifying the live management NIC mid-test risks cutting the provider's own API access; description-only mutation skipped",
	"acc_nvmet_host_subsys_test.go":  "host_id is RequiresReplace; the _update test step is a destroy+create by design, not an in-place Update",
	"acc_nvmet_port_subsys_test.go":  "subsys_id is RequiresReplace; the _update test step is a destroy+create by design, not an in-place Update",
	"acc_pool_test.go":               "data source: pools are managed out-of-band, no Update path through the provider",
	"acc_systemdataset_test.go":      "_update applies same pool value as basic — singleton with one available pool on the test VM, no meaningful mutation",
}

// TestUpdatePlanCheckCoverage scans every acc_*_test.go in
// internal/provider/ that has an _update function and asserts the
// plancheck.ResourceActionUpdate check is present, unless the file
// appears in updatePlanCheckExclusions with rationale.
//
// Without the check, an _update test can pass while silently running
// destroy+create instead of in-place update — exactly the failure
// mode that breaks the day someone accidentally adds RequiresReplace
// to a Required attribute. End-state TestCheck assertions still pass
// because the value is the same after recreate. The plan-shape
// assertion is what catches the regression.
//
// internal/resources/*_test.go _update tests are intentionally NOT
// scanned: those are unit-level wrappers that run the same Update
// code path as the provider-level test. Asserting the plan shape
// once at the canonical acceptance-test layer is sufficient.
func TestUpdatePlanCheckCoverage(t *testing.T) {
	matches, err := filepath.Glob("acc_*_test.go")
	if err != nil {
		t.Fatalf("glob acc tests: %v", err)
	}

	var withCheck, without []string
	for _, f := range matches {
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_invariant_test.go") ||
			strings.HasSuffix(base, "_coverage_test.go") {
			continue
		}
		if _, excluded := updatePlanCheckExclusions[base]; excluded {
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

	total := len(withCheck) + len(without)
	t.Logf("update-plan-shape coverage: %d/%d non-excluded acc tests with _update (%0.1f%%)",
		len(withCheck), total, 100.0*float64(len(withCheck))/float64(total))
	for _, f := range withCheck {
		t.Logf("  OK       %s", f)
	}
	for _, f := range without {
		t.Logf("  MISSING  %s", f)
	}

	if len(without) > 0 {
		t.Fatalf("%d _update acc test(s) missing ResourceActionUpdate plancheck. "+
			"add the assertion to each file, or add the file to "+
			"updatePlanCheckExclusions with a one-line rationale explaining "+
			"why an in-place Update plan cannot be asserted (no-op step, "+
			"RequiresReplace change, etc).\n  missing: %s",
			len(without), strings.Join(without, ", "))
	}
}
