package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// docValidValues matches a documented argument line that enumerates its
// accepted values, e.g.
//
//   - `compression` - (Optional) ... Valid values: `OFF`, `ON`, `LZ4`.
var docValidValues = regexp.MustCompile("(?m)^\\* `([a-z0-9_]+)` -.*?Valid values: (.+?)\\.\\s*$")

// TestDocsValidValuesMatchSchema checks every documented "Valid values" list
// against the resource's own OneOf validator.
//
// These lists are hand-written prose, so they drift silently: widening a
// validator does not touch the doc, and the Registry page is what a
// practitioner reads before deciding a value is unsupported. After the value
// sets were corrected, truenas_dataset still documented GZIP-2 through GZIP-8
// as valid, which no supported TrueNAS has ever accepted, and still capped
// record_size at 1M.
func TestDocsValidValuesMatchSchema(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	docs, err := filepath.Glob(filepath.Join(root, "docs", "resources", "*.md"))
	if err != nil {
		t.Fatalf("globbing docs: %v", err)
	}

	var checked int
	for _, doc := range docs {
		name := strings.TrimSuffix(filepath.Base(doc), ".md")
		resFile := filepath.Join(root, "internal", "resources", name+".go")
		if _, statErr := os.Stat(resFile); statErr != nil {
			continue // documented resource whose file is named differently
		}
		body, readErr := os.ReadFile(doc)
		if readErr != nil {
			t.Errorf("%s: %v", filepath.Base(doc), readErr)
			continue
		}

		for _, m := range docValidValues.FindAllStringSubmatch(string(body), -1) {
			attr, listed := m[1], m[2]
			schemaVals, oneOfErr := oneOfValues(resFile, attr)
			if oneOfErr != nil {
				continue // no OneOf on that attribute; the prose is free-form
			}
			documented := backtickedValues(listed)
			if len(documented) == 0 {
				continue
			}
			checked++

			inSchema := map[string]bool{}
			for _, v := range schemaVals {
				inSchema[v] = true
			}
			inDoc := map[string]bool{}
			for _, v := range documented {
				inDoc[v] = true
			}

			var missing, extra []string
			for _, v := range schemaVals {
				if !inDoc[v] {
					missing = append(missing, v)
				}
			}
			for _, v := range documented {
				if !inSchema[v] {
					extra = append(extra, v)
				}
			}
			sort.Strings(missing)
			sort.Strings(extra)

			if len(missing) > 0 {
				t.Errorf("%s: %s.%s accepts %d value(s) the docs do not list, so the "+
					"Registry page reads as more restrictive than the provider is: %v",
					filepath.Base(doc), name, attr, len(missing), missing)
			}
			if len(extra) > 0 {
				t.Errorf("%s: %s.%s documents %d value(s) the schema rejects, so the "+
					"Registry page advertises something that fails at plan: %v",
					filepath.Base(doc), name, attr, len(extra), extra)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no documented value lists were compared; the walk found nothing")
	}
	t.Logf("compared %d documented value lists against their validators", checked)
}

// backtickedValues pulls the `quoted` items out of a documented list, ignoring
// any unquoted prose around them.
func backtickedValues(s string) []string {
	var out []string
	parts := strings.Split(s, "`")
	for i := 1; i < len(parts); i += 2 {
		if v := strings.TrimSpace(parts[i]); v != "" {
			out = append(out, v)
		}
	}
	return out
}
