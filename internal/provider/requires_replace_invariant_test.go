package provider

// Static invariants that block the phantom-diff anti-pattern from
// creeping back into any resource schema. See each TestXxx for the
// full rationale and the exclusion rules.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// attributeBlockRE matches a single top-level schema attribute declaration
// in a resource file: `"name": schema.XxxAttribute{ ... },`. The capture
// groups are (1) the attribute name and (2) the block body. The body
// match is non-greedy and relies on balanced braces being rare at the
// top level of attribute definitions — good enough for the static
// scan we need here.
var attributeBlockRE = regexp.MustCompile(`(?m)^\s*"(\w+)":\s*schema\.\w+Attribute\{((?:[^{}]|\{[^{}]*\})*?)\}`)

// TestRequiresReplaceRespectsUseStateForUnknown walks every
// internal/resources/*.go file and asserts that any attribute declared
// with BOTH Optional + Computed AND a RequiresReplace() plan modifier
// ALSO has UseStateForUnknown() in its plan modifier slice.
//
// Without that pairing, the Plugin Framework marks the plan value as
// Unknown ("known after apply") on any apply where the user omitted
// the attribute from HCL. RequiresReplace then sees Unknown != state
// value and falsely forces a destroy+create cycle — exactly the bug
// we hit on certificate.key_type during the live TF_ACC run that
// shipped v1.1.0. See internal/resources/certificate.go:key_type for
// the reference pattern.
//
// Acceptable ordering in the PlanModifiers slice: UseStateForUnknown
// must appear BEFORE RequiresReplace. If a custom semantic-equality
// modifier (e.g. PEMEquivalent, JSONEquivalent) is present, it belongs
// between them.
func TestRequiresReplaceRespectsUseStateForUnknown(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}

	type finding struct {
		file string
		attr string
	}
	var gaps []finding

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

		for _, attr := range attributeBlockRE.FindAllStringSubmatch(text, -1) {
			name, body := attr[1], attr[2]
			if !hasField(body, "Optional") || !hasField(body, "Computed") {
				continue
			}
			if !strings.Contains(body, "RequiresReplace()") {
				continue
			}
			if !strings.Contains(body, "UseStateForUnknown()") {
				gaps = append(gaps, finding{file: base, attr: name})
				continue
			}
			// Verify UseStateForUnknown appears before RequiresReplace in the
			// slice — order matters for the framework's plan modifier pipeline.
			useIdx := strings.Index(body, "UseStateForUnknown()")
			repIdx := strings.Index(body, "RequiresReplace()")
			if useIdx > repIdx {
				gaps = append(gaps, finding{
					file: base,
					attr: name + " (UseStateForUnknown must come BEFORE RequiresReplace in the slice)",
				})
			}
		}
	}

	if len(gaps) > 0 {
		var lines []string
		for _, g := range gaps {
			lines = append(lines, "  "+g.file+":"+g.attr)
		}
		t.Fatalf("the following Optional+Computed+RequiresReplace attributes are "+
			"MISSING UseStateForUnknown() (or have it in the wrong slice position) — "+
			"the Plugin Framework will plan them as Unknown on a subsequent apply "+
			"when the user omits them from HCL, and RequiresReplace will then "+
			"falsely force destroy+create:\n%s\n\n"+
			"Fix by adding `stringplanmodifier.UseStateForUnknown()` as the FIRST "+
			"element of the PlanModifiers slice. See internal/resources/certificate.go "+
			"key_type for the reference pattern.",
			strings.Join(lines, "\n"))
	}
}

// hasField returns true if the schema attribute body sets the given
// field to `true`. Handles both aligned (`Optional:    true`) and
// unaligned (`Optional: true`) gofmt spacing.
func hasField(body, name string) bool {
	return strings.Contains(body, name+":") &&
		(strings.Contains(body, name+":    true") ||
			strings.Contains(body, name+": true") ||
			strings.Contains(body, name+":   true") ||
			strings.Contains(body, name+":  true"))
}

// TestOptionalComputedHasUseStateForUnknown is the broader sibling of
// TestRequiresReplaceRespectsUseStateForUnknown: every Optional+Computed
// attribute MUST either (a) carry UseStateForUnknown() in its plan
// modifiers, (b) declare a Default, or (c) appear in the exclusion map
// below with a documented rationale. Without one of those, every
// subsequent `terraform plan` will show the attribute as
// "(known after apply)" — phantom diffs that train operators to ignore
// real drift. This is the #1 cosmetic quality gap that separated the
// v1.0.0 shape from AWS-provider parity and was remediated in Phase C.
//
// The exclusion map holds genuinely-uncommon cases where the server
// regenerates the value on every read (e.g. a timestamp, a counter, a
// computed-at-apply secret). Each entry MUST have a "why" line.
func TestOptionalComputedHasUseStateForUnknown(t *testing.T) {
	// file:attribute -> rationale for skipping UseStateForUnknown.
	// Populate this ONLY when the server legitimately changes the value
	// on every read (counters, live stats, derived fields) OR the
	// attribute appears inside a historical SchemaV0 used exclusively
	// for state migration. Most attributes should NOT be in here — the
	// default behavior is to preserve state when config is null.
	exclusions := map[string]string{
		"cronjob.go:description": "appears in cronjobSchemaV0, the frozen v0 schema used by UpgradeState — historical shape must not be modified or state migration breaks",
	}

	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	var gaps []string
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
		for _, attr := range attributeBlockRE.FindAllStringSubmatch(text, -1) {
			name, body := attr[1], attr[2]
			if name == "id" {
				continue // id always has its own Computed-only path
			}
			if !hasField(body, "Optional") || !hasField(body, "Computed") {
				continue
			}
			if strings.Contains(body, "UseStateForUnknown()") {
				continue
			}
			if strings.Contains(body, "Default:") {
				continue
			}
			key := base + ":" + name
			if _, ok := exclusions[key]; ok {
				continue
			}
			gaps = append(gaps, "  "+key)
		}
	}
	if len(gaps) > 0 {
		t.Fatalf("the following Optional+Computed attributes are missing "+
			"UseStateForUnknown() and have no Default — every `terraform plan` "+
			"will show them as (known after apply) phantom diffs and train "+
			"operators to ignore real drift:\n%s\n\n"+
			"Fix by adding `xplanmodifier.UseStateForUnknown()` (where x is "+
			"string/int64/bool/list/map/set to match the attribute type) to "+
			"the PlanModifiers slice. See internal/resources/dataset.go "+
			"compression for the reference pattern. If the server truly "+
			"regenerates this value on every read, add it to the "+
			"`exclusions` map above WITH A RATIONALE.",
			strings.Join(gaps, "\n"))
	}
}
