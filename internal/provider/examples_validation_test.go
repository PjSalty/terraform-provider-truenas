package provider

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestExamplesResourceHCLValid ensures every examples/resources/truenas_*/resource.tf
// parses as valid HCL syntax. This catches broken example snippets before
// they ship to the Terraform registry. It does not run `terraform plan`,
// so it only checks syntax — not semantic validity against the schema.
func TestExamplesResourceHCLValid(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	examplesDir := filepath.Join(root, "examples", "resources")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no example directories found")
	}

	parser := hclparse.NewParser()
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "truenas_") {
			continue
		}
		rtf := filepath.Join(examplesDir, entry.Name(), "resource.tf")
		if _, err := os.Stat(rtf); err != nil {
			t.Errorf("%s: resource.tf missing", entry.Name())
			continue
		}
		src, err := os.ReadFile(rtf)
		if err != nil {
			t.Errorf("%s: read: %v", entry.Name(), err)
			continue
		}
		_, diags := parser.ParseHCL(src, rtf)
		if diags.HasErrors() {
			t.Errorf("%s: HCL parse errors:\n%s", entry.Name(), diags.Error())
			continue
		}
		count++
	}
	t.Logf("validated %d example resource.tf files", count)
}

// TestExamplesImportShValid ensures every examples/resources/truenas_*/import.sh
// exists and starts with `terraform import`. This enforces that docs generation
// via tfplugindocs can embed the correct import command for every resource.
func TestExamplesImportShValid(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	examplesDir := filepath.Join(root, "examples", "resources")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "truenas_") {
			continue
		}
		sh := filepath.Join(examplesDir, entry.Name(), "import.sh")
		src, err := os.ReadFile(sh)
		if err != nil {
			t.Errorf("%s: import.sh missing or unreadable: %v", entry.Name(), err)
			continue
		}
		// First non-blank, non-comment line must start with `terraform import`.
		found := false
		for _, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if !strings.HasPrefix(trimmed, "terraform import") {
				t.Errorf("%s: first non-comment line must start with 'terraform import', got: %q", entry.Name(), trimmed)
			}
			found = true
			break
		}
		if !found {
			t.Errorf("%s: import.sh has no non-comment content", entry.Name())
		}
	}
}

// TestExamplesAndDocsCoverage ensures every resource registered in the provider
// has both an examples/resources/truenas_<name>/ directory AND a
// docs/resources/<name>.md file. Also verifies no orphan examples exist for
// resources that have been removed from the provider.
func TestExamplesAndDocsCoverage(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}

	// Collect the TypeName of every registered resource.
	p := &TrueNASProvider{}
	ctx := context.Background()
	registered := make(map[string]bool)
	for _, fn := range p.Resources(ctx) {
		r := fn()
		req := resource.MetadataRequest{ProviderTypeName: "truenas"}
		resp := &resource.MetadataResponse{}
		r.Metadata(ctx, req, resp)
		if resp.TypeName == "" {
			t.Errorf("resource %T returned empty TypeName", r)
			continue
		}
		registered[resp.TypeName] = true
	}

	// Walk examples dir.
	examplesDir := filepath.Join(root, "examples", "resources")
	exEntries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples: %v", err)
	}
	exampleDirs := make(map[string]bool)
	for _, e := range exEntries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "truenas_") {
			exampleDirs[e.Name()] = true
		}
	}

	// Walk docs dir.
	docsDir := filepath.Join(root, "docs", "resources")
	docEntries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatalf("reading docs: %v", err)
	}
	docFiles := make(map[string]bool)
	for _, e := range docEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			// docs/resources/<short>.md → truenas_<short>
			short := strings.TrimSuffix(e.Name(), ".md")
			docFiles["truenas_"+short] = true
		}
	}

	// Every registered resource needs an example + doc.
	var missingExamples, missingDocs []string
	for name := range registered {
		if !exampleDirs[name] {
			missingExamples = append(missingExamples, name)
		}
		if !docFiles[name] {
			missingDocs = append(missingDocs, name)
		}
	}
	sort.Strings(missingExamples)
	sort.Strings(missingDocs)
	for _, n := range missingExamples {
		t.Errorf("registered resource %q has no examples/resources/%s/ directory", n, n)
	}
	for _, n := range missingDocs {
		short := strings.TrimPrefix(n, "truenas_")
		t.Errorf("registered resource %q has no docs/resources/%s.md file", n, short)
	}

	// Every example dir and doc file should correspond to a registered resource.
	var orphanExamples, orphanDocs []string
	for name := range exampleDirs {
		if !registered[name] {
			orphanExamples = append(orphanExamples, name)
		}
	}
	for name := range docFiles {
		if !registered[name] {
			orphanDocs = append(orphanDocs, name)
		}
	}
	sort.Strings(orphanExamples)
	sort.Strings(orphanDocs)
	for _, n := range orphanExamples {
		t.Errorf("orphan example dir: examples/resources/%s/ has no registered resource", n)
	}
	for _, n := range orphanDocs {
		short := strings.TrimPrefix(n, "truenas_")
		t.Errorf("orphan doc file: docs/resources/%s.md has no registered resource", short)
	}
}

// findRepoRoot walks up from the package directory to find the go.mod.
func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", os.ErrNotExist
		}
		wd = parent
	}
}
