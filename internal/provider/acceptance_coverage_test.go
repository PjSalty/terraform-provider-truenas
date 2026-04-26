package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// acceptanceCoverageFloor is the minimum number of resource files in
// internal/resources/ that MUST have a corresponding `_test.go` file
// containing at least one `func TestAcc...` function.
//
// This is the Phase J coverage ratchet. A resource without ANY
// acceptance test is a resource we cannot claim works end-to-end —
// Create/Read/Update/Delete, schema validation, plan modifiers,
// config validators, import state, and plan-time diagnostics all
// land on the TestAcc surface in the Plugin Framework's testing
// helper. Missing a TestAcc = missing confidence.
//
// PlanOnly tests (schema validation only, no wire call) count toward
// the floor because they still exercise the full Framework
// Plan phase and fail if the schema, HCL parser, validators, or
// plan modifiers regress. They do NOT catch Create/Update/Delete
// wire-level bugs, but that's the trade-off we accept for resources
// that need special fixture setup to run a real apply.
const acceptanceCoverageFloor = 63

// TestAcceptanceTestCoverage walks internal/resources/*.go, identifies
// every file that declares a resource (via ProviderTypeName + "_..."),
// and for each one verifies that its sibling `*_test.go` file contains
// at least one `func TestAcc...` function.
//
// Fails on:
//   - any resource without a matching _test.go
//   - any _test.go that exists but has no TestAcc function
//   - total coverage dropping below acceptanceCoverageFloor
//
// The test is static: it does NOT execute the acceptance tests
// themselves (TF_ACC=1 handles that). It only verifies structural
// presence, which is cheap (<10ms) and deterministic so prod-ready
// can gate on it with no live infra.
func TestAcceptanceTestCoverage(t *testing.T) {
	resourceFiles := resourceGoFiles(t)
	var missingTestFile, missingTestAccFunc []string
	covered := 0

	for _, rf := range resourceFiles {
		base := strings.TrimSuffix(filepath.Base(rf), ".go")
		testPath := filepath.Join(filepath.Dir(rf), base+"_test.go")
		src, err := os.ReadFile(testPath)
		if err != nil {
			if os.IsNotExist(err) {
				missingTestFile = append(missingTestFile, base)
				continue
			}
			t.Errorf("read %s: %v", testPath, err)
			continue
		}
		if !hasTestAccFunc(src) {
			missingTestAccFunc = append(missingTestAccFunc, base)
			continue
		}
		covered++
	}
	sort.Strings(missingTestFile)
	sort.Strings(missingTestAccFunc)

	if len(missingTestFile) > 0 {
		t.Errorf("resources without a *_test.go sibling file:\n  %s",
			strings.Join(missingTestFile, "\n  "))
	}
	if len(missingTestAccFunc) > 0 {
		t.Errorf("resource *_test.go files without any func TestAcc*:\n  %s",
			strings.Join(missingTestAccFunc, "\n  "))
	}
	if covered < acceptanceCoverageFloor {
		t.Fatalf("acceptance test coverage ratchet regression: have %d "+
			"resources with TestAcc*, want ≥ %d. This is a SLO ratchet — "+
			"if you intentionally removed a resource, lower "+
			"acceptanceCoverageFloor in this test with a PR-comment "+
			"justification.",
			covered, acceptanceCoverageFloor)
	}
	t.Logf("Acceptance test coverage: %d resources with at least one TestAcc (floor %d)",
		covered, acceptanceCoverageFloor)
}

// resourceGoFiles returns the paths of every non-test .go file under
// internal/resources/ that looks like a resource declaration file
// (matches `ProviderTypeName + "_..."` somewhere in the content).
// Files that happen to exist in the resources package but do NOT
// declare a resource (e.g., shared helpers) are filtered out, so
// the ratchet doesn't incorrectly demand a TestAcc for a helper file.
func resourceGoFiles(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	rx := regexp.MustCompile(`ProviderTypeName \+ "_[a-z_]+"`)
	var out []string
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		if rx.Match(src) {
			out = append(out, m)
		}
	}
	sort.Strings(out)
	return out
}

// hasTestAccFunc reports whether the given test file source contains
// at least one `func TestAcc...` top-level declaration. Matches
// TestAccCloudSync_schemaValidation, TestAccDataset_basic, etc.
func hasTestAccFunc(src []byte) bool {
	return regexp.MustCompile(`(?m)^func TestAcc\w*\(`).Match(src)
}
