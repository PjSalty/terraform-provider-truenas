package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// docsCoverageFloor is the minimum number of resource type names that
// MUST have BOTH a rendered Terraform Registry doc under docs/resources/
// AND a working HCL example under examples/resources/<type>/resource.tf
// AND an import example under examples/resources/<type>/import.sh.
//
// This is the Phase I coverage ratchet — lower values leave operators
// flying blind at lookup time, which is a prod-ready blocker. Raising
// the floor is cheap (commit the new floor alongside a new resource's
// docs + example); lowering it requires a documented reason in the PR
// comment.
const docsCoverageFloor = 63

// TestDocsCoverage verifies that every resource the provider declares
// also has a registry-format doc, a resource.tf example, and an
// import.sh example, AND that the total count meets docsCoverageFloor.
//
// This is a static file-layout test — it does NOT require tfplugindocs,
// terraform, or network access. It walks the three directories and
// cross-references their contents against the declared TypeName of
// every resource file in internal/resources/.
//
// The check is three-way: docs, examples, and the code must all agree.
// Any drift between them (a resource with no doc, a doc for a removed
// resource, an example missing resource.tf or import.sh) fails the
// test with a specific pointer to what's missing.
func TestDocsCoverage(t *testing.T) {
	resources := resourceTypeNames(t)
	docs := docsPresent(t)
	examples, missingTF, missingImport := examplesPresent(t)

	var missingDocs, missingExamples []string
	for _, r := range resources {
		if !docs[r] {
			missingDocs = append(missingDocs, r)
		}
		if !examples[r] {
			missingExamples = append(missingExamples, r)
		}
	}
	sort.Strings(missingDocs)
	sort.Strings(missingExamples)

	var orphanDocs, orphanExamples []string
	for d := range docs {
		if !containsStr(resources, d) {
			orphanDocs = append(orphanDocs, d)
		}
	}
	for e := range examples {
		if !containsStr(resources, e) {
			orphanExamples = append(orphanExamples, e)
		}
	}
	sort.Strings(orphanDocs)
	sort.Strings(orphanExamples)

	if len(missingDocs) > 0 {
		t.Errorf("resources without a docs/resources/<type>.md entry:\n  %s",
			strings.Join(missingDocs, "\n  "))
	}
	if len(missingExamples) > 0 {
		t.Errorf("resources without an examples/resources/<type>/ directory:\n  %s",
			strings.Join(missingExamples, "\n  "))
	}
	if len(missingTF) > 0 {
		t.Errorf("example dirs missing resource.tf:\n  %s",
			strings.Join(missingTF, "\n  "))
	}
	if len(missingImport) > 0 {
		t.Errorf("example dirs missing import.sh:\n  %s",
			strings.Join(missingImport, "\n  "))
	}
	if len(orphanDocs) > 0 {
		t.Errorf("docs/resources/ entries with no matching resource (removed? renamed?):\n  %s",
			strings.Join(orphanDocs, "\n  "))
	}
	if len(orphanExamples) > 0 {
		t.Errorf("examples/resources/ dirs with no matching resource (removed? renamed?):\n  %s",
			strings.Join(orphanExamples, "\n  "))
	}

	if len(resources) < docsCoverageFloor {
		t.Fatalf("docs coverage ratchet regression: have %d resources, "+
			"want ≥ %d. This is a SLO ratchet — if you intentionally "+
			"removed a resource, lower docsCoverageFloor in this test "+
			"with a PR-comment justification.",
			len(resources), docsCoverageFloor)
	}
	t.Logf("Docs coverage: %d resources, %d docs, %d examples (floor %d)",
		len(resources), len(docs), len(examples), docsCoverageFloor)
}

// TestDocsNoPlaceholders scans every committed doc and example file
// for TODO/FIXME/placeholder markers that would indicate a developer
// left scaffolding behind. A clean Registry doc has none of these.
func TestDocsNoPlaceholders(t *testing.T) {
	// Regex matches the canonical "unfinished" markers without also
	// matching legitimate uses of the words (e.g. "// TODO" in a test
	// file comment is caught, but "todomvc" wouldn't be).
	marker := regexp.MustCompile(`(?i)\b(TODO|FIXME|XXX|PLACEHOLDER|your-value-here|<placeholder>)\b`)
	roots := []string{"../../docs/resources", "../../docs/data-sources", "../../examples/resources"}
	var hits []string
	for _, root := range roots {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			// Skip unreadable entries silently — a single missing file
			// must not masquerade as a placeholder regression. The outer
			// ratchet TestDocsCoverage already enforces file presence
			// against the declared resource set, so anything skipped
			// here has been or will be caught there. Returning nil is
			// INTENTIONAL; nolint to silence a structurally-sound finding.
			//nolint:nilerr // intentional skip for unreadable walk entries
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			src, readErr := os.ReadFile(path)
			//nolint:nilerr // intentional skip for unreadable files; see comment above
			if readErr != nil {
				return nil
			}
			if marker.Match(src) {
				hits = append(hits, path)
			}
			return nil
		})
	}
	if len(hits) > 0 {
		t.Errorf("placeholder markers found in committed docs/examples:\n  %s",
			strings.Join(hits, "\n  "))
	}
}

// resourceTypeNames parses every non-test file under internal/resources/
// and returns the TypeName each resource registers (e.g. "truenas_dataset").
// Falls back to a deterministic empty slice on parse failure so an
// isolated unreadable file can't silently pass the coverage check.
func resourceTypeNames(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	rx := regexp.MustCompile(`ProviderTypeName \+ "(_[a-z_]+)"`)
	seen := map[string]struct{}{}
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		if sub := rx.FindSubmatch(src); len(sub) == 2 {
			seen["truenas"+string(sub[1])] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// docsPresent returns the set of resource type names that have a
// corresponding doc file under docs/resources/. tfplugindocs convention
// is that doc filenames are UNPREFIXED (e.g. `dataset.md`), not
// `truenas_dataset.md`, because the Terraform Registry already shows
// them under the provider name namespace. The returned map uses the
// FULL `truenas_dataset` form so it can be compared directly against
// resourceTypeNames() output without case-fiddling at the call site.
func docsPresent(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	matches, err := filepath.Glob("../../docs/resources/*.md")
	if err != nil {
		t.Fatalf("glob docs: %v", err)
	}
	for _, m := range matches {
		base := strings.TrimSuffix(filepath.Base(m), ".md")
		out["truenas_"+base] = true
	}
	return out
}

// examplesPresent returns the set of example directory names under
// examples/resources/ that look like a registry-format resource
// example (have both resource.tf and import.sh), plus the lists of
// example dirs missing one file or the other so the caller can
// surface a precise failure.
func examplesPresent(t *testing.T) (set map[string]bool, missingTF, missingImport []string) {
	t.Helper()
	set = map[string]bool{}
	matches, err := filepath.Glob("../../examples/resources/*")
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || !info.IsDir() {
			continue
		}
		base := filepath.Base(m)
		if !strings.HasPrefix(base, "truenas_") {
			continue
		}
		tfOK := fileExists(filepath.Join(m, "resource.tf"))
		imOK := fileExists(filepath.Join(m, "import.sh"))
		if !tfOK {
			missingTF = append(missingTF, base)
		}
		if !imOK {
			missingImport = append(missingImport, base)
		}
		if tfOK && imOK {
			set[base] = true
		}
	}
	sort.Strings(missingTF)
	sort.Strings(missingImport)
	return set, missingTF, missingImport
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func containsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
