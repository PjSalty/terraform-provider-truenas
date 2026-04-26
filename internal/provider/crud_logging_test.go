package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// crudMethodRE matches the opening line of each of the four CRUD method
// signatures in a Plugin Framework resource. The capture group picks up
// the method name (Create, Read, Update, Delete) so the test can report
// gaps precisely. ImportState is intentionally NOT in this list — it is
// a thin wrapper around resource.ImportStatePassthroughID in every file
// and needs no logging of its own.
var crudMethodRE = regexp.MustCompile(`func \([a-z] \*\w+Resource\) (Create|Read|Update|Delete)\(`)

// TestCRUDLogging verifies that every CRUD method in every resource emits
// at least one tflog call. Operator-visible breadcrumbs during plan and
// apply are the difference between "debug this in 30 seconds" and "debug
// this in an hour". The Plugin Framework gives us free trace logging via
// terraform-plugin-log/tflog; there is never a reason to skip it.
//
// The check is scoped tightly: it only looks at the bytes BETWEEN the
// opening `func ... {` and the matching balanced `}`. A top-level
// package-scope tflog call is not enough to satisfy the invariant for a
// given method.
//
// No exclusions: every Plugin Framework CRUD method on every resource
// MUST call tflog.<Something>. Keeping it strict catches drive-by
// refactors that accidentally drop the log and silence the operator.
func TestCRUDLogging(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	type gap struct {
		file   string
		method string
	}
	var gaps []gap

	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		text := string(src)
		for _, match := range crudMethodRE.FindAllStringSubmatchIndex(text, -1) {
			methodName := text[match[2]:match[3]]
			// Find the body of the function: from the `{` that ends the
			// signature to the matching `}` at column 0 ("^}").
			bodyStart := match[1]
			bodyEnd := findFuncBodyEnd(text, bodyStart)
			if bodyEnd == -1 {
				t.Errorf("%s: could not find end of %s method body", base, methodName)
				continue
			}
			body := text[bodyStart:bodyEnd]
			if !strings.Contains(body, "tflog.") {
				gaps = append(gaps, gap{file: base, method: methodName})
			}
		}
	}

	if len(gaps) > 0 {
		lines := make([]string, 0, len(gaps))
		for _, g := range gaps {
			lines = append(lines, "  "+g.file+" "+g.method)
		}
		t.Fatalf("the following CRUD methods do not emit any tflog call — "+
			"operators debugging a plan/apply will have no breadcrumbs:\n%s\n\n"+
			"Fix by adding at least `tflog.Trace(ctx, \"<Method> <Resource> start\")` "+
			"at the top of each CRUD method. See internal/resources/dataset.go for "+
			"the reference pattern.", strings.Join(lines, "\n"))
	}
}

// findFuncBodyEnd returns the index of the matching "\n}\n" (or "\n}" at
// EOF) that closes the function whose body starts at `start`. Relies on
// the gofmt convention that the closing `}` of a top-level function is
// always at column 0 — any nested `}` will be indented. This is simpler
// and more robust than trying to brace-count inside string literals.
func findFuncBodyEnd(text string, start int) int {
	idx := start
	for {
		next := strings.Index(text[idx:], "\n}")
		if next == -1 {
			return -1
		}
		end := idx + next + 2 // include the `}`
		if end == len(text) || text[end] == '\n' {
			return end
		}
		idx = end
	}
}
