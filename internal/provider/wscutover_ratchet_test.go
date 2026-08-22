package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The v2.0 WebSocket cutover invalidated the REST httptest fixtures these
// tests were built on, and they were skipped rather than rewritten. The
// skip message called that "tracked as v2.x polish" while nothing tracked
// it, so the count could grow silently and nobody would know.
//
// It is tracked now: issue #2. This pins the number so it can only go
// down. Lowering the ceiling is a one-line commit alongside each rewrite;
// raising it needs a reason in the diff.
//
// This is deliberately a ceiling and not an exact match. An exact match
// fails on the rewrite commit itself, which trains people to edit the
// number reflexively, and a ratchet nobody reads is not a ratchet.
const wsCutoverSkipCeiling = 40

// wsCutoverSkipCallRE matches a call site, not the helper definition.
var wsCutoverSkipCallRE = regexp.MustCompile(`(?m)^\s*skipWSCutover\(t\)`)

func TestWSCutoverSkipRatchet(t *testing.T) {
	roots := []string{"../provider", "../resources", "../datasources"}

	total := 0
	perRoot := map[string]int{}
	for _, root := range roots {
		matches, err := filepath.Glob(filepath.Join(root, "*_test.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", root, err)
		}
		for _, m := range matches {
			src, err := os.ReadFile(m)
			if err != nil {
				t.Fatalf("read %s: %v", m, err)
			}
			n := len(wsCutoverSkipCallRE.FindAllString(string(src), -1))
			total += n
			perRoot[strings.TrimPrefix(root, "../")] += n
		}
	}

	if total > wsCutoverSkipCeiling {
		t.Errorf("skipWSCutover call sites grew to %d, ceiling is %d (per package: %v).\n\n"+
			"These are unit tests switched off by the v2.0 WebSocket cutover, tracked in issue #2.\n"+
			"Do not add more. Write the test against wsclient.NewTestServer / newWSTestClient,\n"+
			"which is what the surviving tests use.",
			total, wsCutoverSkipCeiling, perRoot)
	}

	// A ceiling that has drifted far above reality stops being a ratchet:
	// it silently permits regrowth back up to the stale number. Lower it in
	// the same commit that removes the skips.
	if total < wsCutoverSkipCeiling {
		t.Errorf("skipWSCutover call sites are down to %d but the ceiling is still %d "+
			"(per package: %v).\n\nLower wsCutoverSkipCeiling to %d in this commit, "+
			"otherwise the ratchet permits regrowth back to %d.",
			total, wsCutoverSkipCeiling, perRoot, total, wsCutoverSkipCeiling)
	}
}
