package provider

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// sweepFuncRE matches the signature of every sweep function:
//
//	func sweepXxx(_ string) error {
//
// The captured group is the function name (sweepXxx). The plugin-
// testing framework calls every registered sweeper with a region
// string; ours all ignore it because we have one TrueNAS to clean.
var sweepFuncRE = regexp.MustCompile(`(?m)^func (sweep[A-Z]\w*)\(_ string\) error \{`)

// sweepPrefixGuardRE matches the helper calls every sweeper uses to
// scope deletions to fixtures created by acceptance tests:
//
//	sweeperHasAcctestPrefix(name)
//	sweeperDatasetIsAcctest(id)
//	sweep.HasAcctestPrefix(name)
//
// Word boundaries are intentionally not required — the helper names
// in this tree all follow the "Xxx[Is|Has]Acctest" pattern, and the
// substring "Acctest" (capitalised) is unique enough across the test
// tree that loose matching is safe.
var sweepPrefixGuardRE = regexp.MustCompile(`(?:[Ii]s|[Hh]as)[Aa]cctest`)

// sweepPrefixOptOutRE matches an explicit per-sweeper comment that
// the sweeper is exempt from the prefix-guard requirement. Form:
//
//	// sweep-no-prefix-guard: <one-line reason>
//
// Examples where this is legitimate: a singleton config sweeper
// that resets a field to default, not delete by name. A sweeper
// that filters by some other safe attribute (e.g. tag range,
// reserved ID range).
var sweepPrefixOptOutRE = regexp.MustCompile(`(?m)//\s*sweep-no-prefix-guard:\s*\S+`)

// TestSweepersHaveAcctestPrefixGuard verifies that every sweep<Name>
// function in sweeper_test.go either calls an Acctest-prefix helper
// or carries an explicit sweep-no-prefix-guard opt-out comment.
//
// Why: a sweeper without a prefix guard deletes EVERY upstream
// resource of its kind that exists on the target TrueNAS. That's
// catastrophic on any system that holds real data — exactly the
// "we ran the sweeper against prod by accident" failure mode that
// makes operators afraid to use sweepers at all.
//
// The TRUENAS_PROD_DENY safety rail in internal/acctest blocks the
// production hostname today, but the static invariant defends in
// depth: even if someone disables the deny-list, the sweepers
// themselves still refuse to delete anything not matching the
// acceptance-test fixture prefix.
func TestSweepersHaveAcctestPrefixGuard(t *testing.T) {
	src, err := os.ReadFile("sweeper_test.go")
	if err != nil {
		t.Fatalf("read sweeper_test.go: %v", err)
	}
	body := string(src)

	// Find every sweep function and the slice of source between its
	// opening brace and the next top-level closing brace.
	funcMatches := sweepFuncRE.FindAllStringSubmatchIndex(body, -1)
	if len(funcMatches) == 0 {
		t.Fatal("no sweep<Name> functions found in sweeper_test.go — regex broken?")
	}

	type fnSpan struct {
		name string
		body string
	}
	var fns []fnSpan
	for i, m := range funcMatches {
		start := m[0]
		end := len(body)
		if i+1 < len(funcMatches) {
			end = funcMatches[i+1][0]
		}
		nameStart := m[2]
		nameEnd := m[3]
		fns = append(fns, fnSpan{
			name: body[nameStart:nameEnd],
			body: body[start:end],
		})
	}

	var missing []string
	for _, f := range fns {
		hasGuard := sweepPrefixGuardRE.MatchString(f.body)
		hasOptOut := sweepPrefixOptOutRE.MatchString(f.body)
		if !hasGuard && !hasOptOut {
			missing = append(missing, f.name)
		}
	}

	if len(missing) > 0 {
		t.Fatalf("%d sweeper(s) missing a prefix guard — every sweeper MUST either "+
			"call an Acctest-prefix helper (sweeperHasAcctestPrefix, "+
			"sweeperDatasetIsAcctest, etc.) OR carry an explicit "+
			"// sweep-no-prefix-guard: <reason> comment. an unscoped sweeper "+
			"deletes EVERY upstream resource of its kind, which is catastrophic "+
			"on any system holding real data. offenders:\n  - %s",
			len(missing), strings.Join(missing, "\n  - "))
	}

	t.Logf("OK: %d sweepers all have prefix guards or documented opt-outs", len(fns))
}
