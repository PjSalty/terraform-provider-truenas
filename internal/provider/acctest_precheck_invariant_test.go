package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// testAccFuncRE matches every acceptance test function in the resources
// and datasources packages. The convention is fixed across the tree:
//
//	func TestAccDataset_basic(t *testing.T) {
//
// We capture the name so error messages can report the offending test.
var testAccFuncRE = regexp.MustCompile(`(?m)^func (TestAcc\w+)\(t \*testing\.T\) \{`)

// precheckOrSkipRE matches the two valid prologues for an acceptance
// test: either it wires the standard PreCheck callback (which guards
// against running without TRUENAS_URL/TRUENAS_API_KEY), or it stubs
// out the entire body with t.Skip (used for tests that need extra
// infrastructure like dedicated disks).
//
// The check is intentionally permissive on placement — PreCheck and
// t.Skip can appear anywhere in the function body, not just on the
// first line. The goal is to catch tests that forgot both, not to
// police where they are.
var precheckOrSkipRE = regexp.MustCompile(`(PreCheck:|t\.Skip\()`)

// TestAcceptanceTestsHavePreCheckOrSkip verifies every TestAcc* in the
// resources and datasources packages either registers a PreCheck
// callback or short-circuits via t.Skip(). Without one of the two,
// running `go test ./... -run TestAcc` on a workstation that does not
// have TRUENAS_URL / TRUENAS_API_KEY set produces confusing dial
// failures partway through CRUD steps instead of a clean skip.
//
// PreCheck is the standard major-provider pattern (terraform-provider-
// aws, terraform-provider-google, terraform-provider-cloudflare all
// gate every TestAcc behind one). t.Skip is the alternative for tests
// that need infrastructure beyond what PreCheck can validate (e.g.
// TRUENAS_TEST_POOL_DISKS for pool creation).
//
// Without this invariant a contributor can add a new TestAcc that
// silently bypasses the gating and starts failing on developer
// workstations that lack the test VM credentials — exactly the kind
// of "ran on my machine but flakes in CI" pattern this provider's
// other invariants are designed to prevent.
func TestAcceptanceTestsHavePreCheckOrSkip(t *testing.T) {
	roots := []string{"../resources", "../datasources"}
	var missing []string
	for _, root := range roots {
		matches, err := filepath.Glob(root + "/*_test.go")
		if err != nil {
			t.Fatalf("glob %s: %v", root, err)
		}
		for _, m := range matches {
			src, err := os.ReadFile(m)
			if err != nil {
				t.Fatalf("read %s: %v", m, err)
			}
			text := string(src)
			base := filepath.Base(m)
			for _, fn := range testAccFuncRE.FindAllStringSubmatch(text, -1) {
				name := fn[1]
				body := extractFunctionBody(text, fn[0])
				if body == "" {
					// Defensive — extractFunctionBody returns "" only
					// if the function couldn't be located, which would
					// indicate a malformed source file. Skip rather
					// than fail.
					continue
				}
				if !precheckOrSkipRE.MatchString(body) {
					missing = append(missing, base+":"+name)
				}
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("the following acceptance tests neither register a PreCheck "+
			"callback nor short-circuit via t.Skip(). Running the test suite "+
			"on a workstation that lacks TRUENAS_URL / TRUENAS_API_KEY will "+
			"produce dial failures instead of clean skips:\n  %s\n\n"+
			"Fix by either:\n"+
			"  (a) Adding the standard PreCheck callback:\n"+
			"      PreCheck: func() { testAccPreCheck(t) },\n"+
			"  (b) Or stubbing the test with a t.Skip() if it needs extra\n"+
			"      infrastructure beyond what PreCheck can validate:\n"+
			"      t.Skip(\"requires X; enable with TF_ACC=1 and ENVVAR=Y\")\n",
			strings.Join(missing, "\n  "))
	}
}

// extractFunctionBody returns the source between the opening `{` of
// the function whose signature matches `signature` and the matching
// closing `}`. Returns "" if the signature is not found or the braces
// are unbalanced.
//
// We use brace-matching rather than a Go AST parser because the
// invariant tests in this package have stuck with regex+string for
// simplicity — see registration_invariant_test.go for the precedent.
func extractFunctionBody(src, signature string) string {
	idx := strings.Index(src, signature)
	if idx < 0 {
		return ""
	}
	start := idx + len(signature)
	depth := 1
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start:i]
			}
		}
	}
	return ""
}
