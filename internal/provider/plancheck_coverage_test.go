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

// idempotencyExclusions lists every acc_*_test.go file that does NOT
// need a PostApplyPostRefresh ExpectEmptyPlan check, with rationale.
// EVERY other file in internal/provider/acc_*_test.go MUST carry the
// idempotency check. There is no floor, the contract is 100% of the
// non-excluded surface.
//
// Categories of legitimate exclusion:
//   - PlanOnly negative-path tests (acc_validator_errors_test.go), there
//     is no Apply step, so PostApplyPostRefresh is structurally N/A.
//   - Helpers (acc_helpers_test.go), not a test file, just shared
//     scaffolding.
//   - Data-source-only resources (acc_app_test.go uses the catalog
//     data source; acc_pool_test.go is read-only on a pre-existing
//     pool).
//   - Singletons / data sources / read-only resources where the
//     basic test is a PlanOnly check or otherwise lacks an Apply
//     step that could be followed by a refresh.
//
// Adding to this list is a deliberate, reviewed act, the rationale
// MUST explain why the resource genuinely cannot carry the check, not
// "it failed when I ran it". A failure is exactly the bug the check is
// designed to surface; the fix goes in the resource code, not the
// exclusion list.
var idempotencyExclusions = map[string]string{
	"acc_helpers_test.go":           "scaffolding: not a test file",
	"acc_validator_errors_test.go":  "PlanOnly: every test asserts validator rejection before Apply, no refresh to compare against",
	"acc_app_test.go":               "beta data-source style: only exercises truenas_catalog data source, no managed-resource Apply path",
	"acc_pool_test.go":              "data source: reads an existing pool via truenas_pool, no managed-resource lifecycle",
	"acc_network_interface_test.go": "import-only: basic test is ImportState against a pre-existing NIC, no Apply step (modifying the live management interface risks cutting the provider's own API access)",
}

// TestIdempotencyCheckCoverage walks every internal/provider/acc_*_test.go
// file and asserts that the apply-idempotency check is present, unless
// the file appears in idempotencyExclusions with rationale. This is the
// post-rigor-batch contract: no floor, 100% of the non-excluded surface.
func TestIdempotencyCheckCoverage(t *testing.T) {
	matches, err := filepath.Glob("acc_*_test.go")
	if err != nil {
		t.Fatalf("glob acc tests: %v", err)
	}

	var withCheck, without []string
	for _, f := range matches {
		base := filepath.Base(f)
		// Skip invariant tests, they're static-analysis tools that
		// happen to start with acc_, not actual acceptance tests.
		if strings.HasSuffix(base, "_invariant_test.go") {
			continue
		}
		if _, excluded := idempotencyExclusions[base]; excluded {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if bytes.Contains(src, []byte("PostApplyPostRefresh")) &&
			bytes.Contains(src, []byte("plancheck.ExpectEmptyPlan")) {
			withCheck = append(withCheck, base)
		} else {
			without = append(without, base)
		}
	}
	sort.Strings(withCheck)
	sort.Strings(without)

	total := len(withCheck) + len(without)
	t.Logf("apply-idempotency coverage: %d/%d non-excluded acc tests (%0.1f%%)",
		len(withCheck), total, 100.0*float64(len(withCheck))/float64(total))
	for _, f := range withCheck {
		t.Logf("  OK       %s", f)
	}
	for _, f := range without {
		t.Logf("  MISSING  %s", f)
	}

	if len(without) > 0 {
		t.Fatalf("%d acc test(s) missing PostApplyPostRefresh: ExpectEmptyPlan. "+
			"add the check to each file, or add the file to idempotencyExclusions "+
			"with a one-line rationale explaining why the apply-idempotency "+
			"invariant cannot hold for it. \"the test fails when I run it\" is "+
			"NOT a valid exclusion reason, that is the bug the check exists to "+
			"surface, and the fix goes in the resource code.\n  missing: %s",
			len(without), strings.Join(without, ", "))
	}
}
