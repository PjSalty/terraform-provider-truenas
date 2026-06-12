package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// validatorErrorTestRE matches every TestAccValidator_* function in
// the acc_validator_errors_test.go file. The naming convention is
// deliberately distinct from regular acceptance tests so the static
// scanner can count them without false positives.
var validatorErrorTestRE = regexp.MustCompile(`(?m)^func (TestAccValidator_\w+)\(t \*testing\.T\)`)

// validatorErrorTestMinimum is the floor for negative-path validator
// tests in acc_validator_errors_test.go. Every wired validator should
// have at least one ExpectError test asserting plan-time rejection of
// hostile input. The floor only goes up.
//
// Adding a new validator? Land it with at least one TestAccValidator_*
// test exercising the failure path, then bump this number to match.
// Removing one of these tests would silently drop a negative-path
// guarantee, never do it without also removing the validator itself.
const validatorErrorTestMinimum = 22

// TestValidatorErrorCoverage counts the TestAccValidator_* functions
// in acc_validator_errors_test.go and asserts the count is at least
// validatorErrorTestMinimum. Acts as a ratchet so future commits can
// add validators only by also adding their negative-path tests.
//
// Why a separate ratchet vs lumping in with the idempotency
// coverage check: idempotency is a per-resource concern, validator
// error coverage is per-validator. Separating the two ratchets keeps
// each commit's intent obvious to reviewers.
func TestValidatorErrorCoverage(t *testing.T) {
	path := filepath.Join(".", "acc_validator_errors_test.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	matches := validatorErrorTestRE.FindAllStringSubmatch(string(src), -1)
	got := len(matches)

	t.Logf("validator-error coverage: %d TestAccValidator_* functions", got)
	for _, m := range matches {
		t.Logf("  - %s", m[1])
	}

	if got < validatorErrorTestMinimum {
		t.Fatalf("validator-error coverage regressed: %d TestAccValidator_* "+
			"functions, want at least %d (validatorErrorTestMinimum). adding a "+
			"new validator without an ExpectError test removes a plan-time guarantee. "+
			"either restore the test or, if a validator was removed, drop the "+
			"corresponding test and lower the ratchet.",
			got, validatorErrorTestMinimum)
	}
}
