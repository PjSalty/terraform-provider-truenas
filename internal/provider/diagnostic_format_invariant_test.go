package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// addErrorRE matches every Diagnostics.AddError call's summary string
// , the first argument. The pattern handles a single-line summary
// (the dominant style in this codebase); multi-line summaries are
// flagged manually if they appear because they're rare and worth a
// human review.
var addErrorRE = regexp.MustCompile(`(?m)\.AddError\(\s*"([^"]+)"`)

// addAttributeErrorRE is the same for AddAttributeError, which uses
// the same summary convention.
var addAttributeErrorRE = regexp.MustCompile(`(?m)\.AddAttributeError\(\s*[^,]+,\s*"([^"]+)"`)

// allowedSummaryPatterns is the canonical set of diagnostic summary
// shapes the provider ships. Anything outside this set is rejected by
// the invariant unless added to the exemption map with rationale.
//
// Shapes:
//   - "Invalid <Attribute>"            (validation errors on a field)
//   - "Error <Verbing> <Resource>"     (CRUD operation failures)
//   - "Could not <verb> <Resource>: …" (parsing / setup failures)
//   - "Unable to <verb> <Resource>"    (capability-style failures)
//   - "Conflicting <fields>"           (mutually-exclusive attr errors)
//   - "Incomplete <field> mapping"     (incomplete config validation)
//   - "<field> must <constraint>"      (cross-attribute constraints)
//
// Each pattern starts with a capital letter. The TrueNAS-specific
// acronyms iSCSI, NVMe-oF, SMB, NFS, ZFS, ACL, KMIP, SOPS, AD/LDAP/IPA
// are recognised verbatim, they're industry conventions that the
// upstream API also uses in its own error messages.
var allowedSummaryPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^Invalid [A-Za-z_]`),
	regexp.MustCompile(`^Error [A-Z][a-z]+(ing|ed) [A-Za-z]`),
	regexp.MustCompile(`^Could not [a-z]+ [A-Za-z]`),
	regexp.MustCompile(`^Unable to [a-z]+ [A-Za-z]`),
	regexp.MustCompile(`^Conflicting [a-z_]`),
	regexp.MustCompile(`^Incomplete [a-z_]+ (mapping|config|attribute)`),
	regexp.MustCompile(`^[a-z_]+ must [a-z]`), // "<field> must be less than ..."
	regexp.MustCompile(`^Configuring [A-Z]`),
	regexp.MustCompile(`^Missing `),
	regexp.MustCompile(`^Unexpected `),
	regexp.MustCompile(`^Unsupported `),
	regexp.MustCompile(`^Invalid$`),
}

// allowedNonCanonicalSummaries lists known summaries that don't match
// the canonical shapes above but ship for a deliberate reason. Each
// entry needs a rationale comment so the next reviewer can decide
// whether the exemption is still justified.
var allowedNonCanonicalSummaries = map[string]string{
	"Validation Error":           "framework convention, used by per-attribute validators",
	"Resource Not Found":         "explicit not-found surface; doesn't fit 'Error <verbing>' shape",
	"Required attribute missing": "framework convention, emitted by RequiredWhenEqual validator",
	"Disk Not Found":             "datasource lookup-by-name surface; '<Resource> Not Found' is the framework convention for this case",
	"Pool Not Found":             "datasource lookup-by-name surface; '<Resource> Not Found' is the framework convention for this case",
}

// TestDiagnosticFormat_AddErrorSummaries scans every resource and
// data-source source file for AddError / AddAttributeError calls and
// asserts each summary follows one of the canonical shapes. A drift -
// "error creating X" (lowercase verb), "failed to do Y" (different
// pattern), "Something went wrong" (no resource name), surfaces as
// a per-call failure with the exact file + summary.
//
// Why this matters at major-provider rigor: terraform plan output
// reads diagnostic summaries directly. Inconsistent capitalization
// or pattern reads as carelessness; consistent shapes give the
// operator a quick mental model for what kind of failure they're
// looking at.
func TestDiagnosticFormat_AddErrorSummaries(t *testing.T) {
	files, err := schemaFiles()
	if err != nil {
		t.Fatalf("enumerate files: %v", err)
	}

	type finding struct {
		file    string
		summary string
	}
	var bad []finding

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(data)
		base := labelForFile(f)

		for _, re := range []*regexp.Regexp{addErrorRE, addAttributeErrorRE} {
			matches := re.FindAllStringSubmatch(src, -1)
			for _, m := range matches {
				summary := m[1]
				// Strip format directives so e.g. "Could not parse X ID: %s"
				// reads the same as "Could not parse X ID".
				normalised := stripFormatVerbs(summary)
				if matchesAny(allowedSummaryPatterns, normalised) {
					continue
				}
				if _, ok := allowedNonCanonicalSummaries[summary]; ok {
					continue
				}
				bad = append(bad, finding{file: base, summary: summary})
			}
		}
	}

	if len(bad) > 0 {
		// Dedupe, the same summary often appears multiple times.
		seen := map[string]string{}
		for _, b := range bad {
			key := b.file + "::" + b.summary
			seen[key] = b.summary
		}
		keys := make([]string, 0, len(seen))
		for k := range seen {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var sb strings.Builder
		sb.WriteString("Diagnostic summaries that don't match the canonical shapes:\n\n")
		for _, k := range keys {
			sb.WriteString("  ")
			sb.WriteString(k)
			sb.WriteString("\n")
		}
		sb.WriteString("\nCanonical shapes:\n")
		sb.WriteString("  - \"Invalid <Attribute>\"\n")
		sb.WriteString("  - \"Error <Verbing> <Resource>\"\n")
		sb.WriteString("  - \"Could not <verb> <Resource>: ...\"\n")
		sb.WriteString("  - \"Unable to <verb> <Resource>\"\n")
		sb.WriteString("\nUpdate the summary to match one of the shapes, OR add the\n")
		sb.WriteString("verbatim summary to allowedNonCanonicalSummaries with a rationale.\n")
		t.Error(sb.String())
	}
}

// stripFormatVerbs removes fmt-style %s / %d / %v from a summary so
// the canonical match doesn't get tripped up by parametric values.
func stripFormatVerbs(s string) string {
	s = regexp.MustCompile(`%[sdvqxX]`).ReplaceAllString(s, "X")
	s = regexp.MustCompile(`%\d+\w`).ReplaceAllString(s, "X")
	return strings.TrimSpace(s)
}

func matchesAny(res []*regexp.Regexp, s string) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// Use filepath.Glob via the shared helper but also pull this file's
// import live for compilers that complain otherwise.
var _ = filepath.Glob
