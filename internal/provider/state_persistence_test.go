package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// stateWritingMethodRE matches the opening line of Create / Read / Update
// method signatures. These are the methods that MUST end by writing state
// back via resp.State.Set(ctx, &model) (or an equivalent) — without it,
// the state file does not reflect the real API response and the next
// plan reports a phantom diff on every operation. Delete does NOT write
// state (the Plugin Framework clears it automatically on Delete success),
// so it is not included here.
var stateWritingMethodRE = regexp.MustCompile(`func \([a-z] \*\w+Resource\) (Create|Read|Update)\(`)

// TestStatePersistence verifies that every Create/Read/Update method
// in every resource file writes state back via a resp.State.Set call
// (or the rarer Diagnostics.Append(resp.State.Set(...)...) wrapper).
// Missing state persistence on these paths is the failure mode that
// turns "apply works" into "plan never shows clean" — the classic
// provider anti-pattern that the plancheck ExpectEmptyPlan ratchet
// also defends against, at a different layer.
//
// Singleton resources that do not extract an ID but still call State.Set
// with the model are fine — the check only requires the bytes
// `resp.State.Set(` or `.State.Set(ctx` to appear in the function body,
// which every Plugin Framework idiom uses.
func TestStatePersistence(t *testing.T) {
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
		for _, match := range stateWritingMethodRE.FindAllStringSubmatchIndex(text, -1) {
			methodName := text[match[2]:match[3]]
			bodyStart := match[1]
			bodyEnd := findFuncBodyEnd(text, bodyStart)
			if bodyEnd == -1 {
				t.Errorf("%s: could not find end of %s method body", base, methodName)
				continue
			}
			body := text[bodyStart:bodyEnd]
			if !strings.Contains(body, "resp.State.Set(") &&
				!strings.Contains(body, ".State.Set(ctx") {
				gaps = append(gaps, gap{file: base, method: methodName})
			}
		}
	}

	if len(gaps) > 0 {
		lines := make([]string, 0, len(gaps))
		for _, g := range gaps {
			lines = append(lines, "  "+g.file+" "+g.method)
		}
		t.Fatalf("the following Create/Read/Update methods do not call resp.State.Set — "+
			"the mutation will not be persisted to state, and the next plan will "+
			"report a phantom diff forever:\n%s\n\n"+
			"Fix by adding `resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)` "+
			"at the end of the method body. See internal/resources/dataset.go Create "+
			"for the reference pattern.", strings.Join(lines, "\n"))
	}
}
