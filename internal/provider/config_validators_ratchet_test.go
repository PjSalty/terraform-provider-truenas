package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configValidatorsFloor is the minimum number of resource files that
// MUST declare a ConfigValidators method. This is a ratchet: raising
// the floor is cheap (commit the new floor alongside the new validator),
// lowering it requires justification in a PR comment. The mechanism is
// identical to TestIdempotencyCheckCoverage — a SLO-style gate that
// prevents the coverage from silently regressing when someone copies
// a resource template that omitted its ConfigValidators hook.
//
// Bump this number every time a new resource file adds a
// ConfigValidators() method. Never lower it without documented reason.
const configValidatorsFloor = 3

// TestConfigValidatorsCoverage counts the number of resource files
// that declare a `func (r *XxxResource) ConfigValidators(` method and
// fails if the count drops below configValidatorsFloor. Resources that
// have an enum-style discriminator (create_type, type, protocol,
// auth_method, mode) are the prime candidates — see
// docs/guides/phased-rollout.md for the reference pattern and
// internal/resources/certificate.go for the first implementation.
func TestConfigValidatorsCoverage(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	count := 0
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
		if strings.Contains(string(src), ") ConfigValidators(") {
			count++
			files = append(files, base)
		}
	}
	if count < configValidatorsFloor {
		t.Fatalf("ConfigValidators coverage dropped below floor: have %d, "+
			"want ≥ %d. Files with ConfigValidators: %v\n\n"+
			"This is a SLO-style ratchet. Either add a ConfigValidators hook "+
			"to at least one more resource (preferred — raises the floor) "+
			"or, if you intentionally removed one, lower configValidatorsFloor "+
			"in this file with a PR-comment justification.",
			count, configValidatorsFloor, files)
	}
	t.Logf("ConfigValidators coverage: %d resource files (floor %d). Files: %v",
		count, configValidatorsFloor, files)
}
