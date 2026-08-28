package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	fuzzReadmeRow = regexp.MustCompile(`(?m)^\| (internal/[a-z]+) +\| +(\d+) \|`)
	// Only the names inside a -fuzz pattern, which is where staleness bites:
	// the placeholder <FuzzName> in the corpus-path sentence and this file's
	// own test names are prose, not targets.
	fuzzReadmeName = regexp.MustCompile(`-fuzz='\^(Fuzz[A-Za-z0-9_]+)\$'`)
	fuzzFuncDecl   = regexp.MustCompile(`(?m)^func (Fuzz[A-Za-z0-9_]+)\(`)
)

// TestFuzzReadmeMatchesTargets holds testdata/fuzz/README.md to the code.
//
// It used to name FuzzParseRetryAfter in internal/client and give a smoke-run
// command against that package. Both were deleted in the v2.0 WebSocket
// cutover, so the one command the document offered could not run, and its
// table listed 8 targets where the repo has 101.
func TestFuzzReadmeMatchesTargets(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "testdata", "fuzz", "README.md"))
	if err != nil {
		t.Fatalf("reading the fuzz README: %v", err)
	}

	// Every Fuzz name the document mentions has to exist.
	declared := map[string]string{} // target -> package
	pkgs, err := filepath.Glob(filepath.Join(root, "internal", "*"))
	if err != nil {
		t.Fatalf("globbing internal: %v", err)
	}
	counts := map[string]int{}
	for _, pkg := range pkgs {
		files, globErr := filepath.Glob(filepath.Join(pkg, "*_test.go"))
		if globErr != nil {
			t.Fatalf("globbing %s: %v", pkg, globErr)
		}
		rel := "internal/" + filepath.Base(pkg)
		for _, f := range files {
			src, readErr := os.ReadFile(f)
			if readErr != nil {
				t.Fatalf("reading %s: %v", f, readErr)
			}
			for _, m := range fuzzFuncDecl.FindAllStringSubmatch(string(src), -1) {
				declared[m[1]] = rel
				counts[rel]++
			}
		}
	}
	if len(declared) == 0 {
		t.Fatal("no fuzz targets found; the walk is broken, not the README")
	}

	named := fuzzReadmeName.FindAllStringSubmatch(string(body), -1)
	if len(named) == 0 {
		t.Fatal("the fuzz README shows no -fuzz pattern to check")
	}
	for _, m := range named {
		if _, ok := declared[m[1]]; !ok {
			t.Errorf("the fuzz README names %s, which no longer exists", m[1])
		}
	}

	// And every count in its table has to be the real one.
	rows := fuzzReadmeRow.FindAllStringSubmatch(string(body), -1)
	if len(rows) == 0 {
		t.Fatal("the fuzz README has no package table rows to check")
	}
	inTable := map[string]bool{}
	for _, r := range rows {
		pkg, want := r[1], r[2]
		inTable[pkg] = true
		n, convErr := strconv.Atoi(want)
		if convErr != nil {
			t.Errorf("%s: unreadable count %q", pkg, want)
			continue
		}
		if counts[pkg] != n {
			t.Errorf("the fuzz README says %s has %d targets, it has %d", pkg, n, counts[pkg])
		}
	}
	for pkg, n := range counts {
		if n > 0 && !inTable[pkg] {
			t.Errorf("%s has %d fuzz targets and no row in the fuzz README", pkg, n)
		}
	}
	t.Logf("checked %d fuzz targets across %d packages, %d named in commands", len(declared), len(rows), len(named))
}

// The smoke-run command in the README has to name a package that exists, which
// is the specific way the old one rotted.
func TestFuzzReadmeCommandsPointAtRealPackages(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "testdata", "fuzz", "README.md"))
	if err != nil {
		t.Fatalf("reading the fuzz README: %v", err)
	}
	re := regexp.MustCompile(`\./(internal/[a-z]+)/`)
	found := re.FindAllStringSubmatch(string(body), -1)
	if len(found) == 0 {
		t.Fatal("the fuzz README shows no go test command to check")
	}
	for _, m := range found {
		if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(m[1]))); statErr != nil {
			t.Errorf("the fuzz README runs against %s, which does not exist", m[1])
		}
	}
	if strings.Contains(string(body), "internal/client") {
		t.Error("the fuzz README still references internal/client, deleted in the v2.0 cutover")
	}
}
