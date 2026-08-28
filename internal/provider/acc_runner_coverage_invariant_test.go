package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	accRunnerLine = regexp.MustCompile(`(?m)^TF_ACC=1 go test .*?(\./internal/\S+(?:\s+\./internal/\S+)*)\s*2>&1`)
	accFuncDecl   = regexp.MustCompile(`(?m)^func (TestAcc[A-Za-z0-9_]*)\(`)
)

// TestAccPackagesAreInAccRunner fails when a package containing acceptance
// tests is missing from the package list scripts/acc.sh actually runs.
//
// internal/provider carried 192 TestAcc functions and was not on that line, so
// `make acc`, the pipeline whose own banner calls it the full acceptance
// suite, silently ran none of them. They were reachable only through
// `make testacc`, which nothing in the documented flow calls. A test that
// exists and never runs reads exactly like a passing one.
func TestAccPackagesAreInAccRunner(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	script, err := os.ReadFile(filepath.Join(root, "scripts", "acc.sh"))
	if err != nil {
		t.Fatalf("reading scripts/acc.sh: %v", err)
	}
	m := accRunnerLine.FindStringSubmatch(string(script))
	if m == nil {
		t.Fatal("could not find the acceptance `go test` line in scripts/acc.sh")
	}
	inRunner := map[string]bool{}
	for _, p := range strings.Fields(m[1]) {
		inRunner[strings.TrimSuffix(strings.TrimPrefix(p, "./"), "/")] = true
	}
	if len(inRunner) == 0 {
		t.Fatal("the acceptance runner names no packages")
	}

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
		for _, f := range files {
			src, readErr := os.ReadFile(f)
			if readErr != nil {
				t.Fatalf("reading %s: %v", f, readErr)
			}
			counts["internal/"+filepath.Base(pkg)] += len(accFuncDecl.FindAllString(string(src), -1))
		}
	}

	var missing []string
	total := 0
	for pkg, n := range counts {
		if n == 0 {
			continue
		}
		total += n
		if !inRunner[pkg] {
			missing = append(missing, pkg)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("packages with acceptance tests that scripts/acc.sh does not run: %v\n"+
			"Add them to the `go test` package list there, or the suite reports success "+
			"without having run them.", missing)
	}
	if total == 0 {
		t.Fatal("no acceptance tests found at all; the walk is broken, not the runner")
	}
	t.Logf("checked %d acceptance tests across %d packages", total, len(counts))
}

// The acceptance packages share ONE appliance and ten singleton config
// resources are exercised from both of them, so the runner must not run
// packages concurrently.
func TestAccRunnerIsSerial(t *testing.T) {
	t.Parallel()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("finding repo root: %v", err)
	}
	// Checked against the ARGUMENT LIST, not the file. A first cut looked for
	// "-p 1" anywhere in the file and passed on a file where the only match was
	// the comment explaining why the flag is there.
	for _, c := range []struct{ rel, marker string }{
		{"scripts/acc.sh", "ACCEPTANCE_ARGS=("},
		{"Makefile", "TF_ACC=1 $(GO) test"},
	} {
		rel := c.rel
		body, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			t.Fatalf("reading %s: %v", rel, readErr)
		}
		i := strings.Index(string(body), c.marker)
		if i < 0 {
			t.Fatalf("%s: could not find %q to read the acceptance args from", rel, c.marker)
		}
		args := string(body)[i:]
		if j := strings.Index(args, "\n\n"); j > 0 {
			args = args[:j]
		}
		if !strings.Contains(args, "-p 1") {
			t.Errorf("%s runs acceptance tests without -p 1; the two acceptance "+
				"packages both manage smb_config, nfs_config, ssh_config, snmp_config, "+
				"mail_config, ftp_config, systemdataset, network_config, ups_config and "+
				"kmip_config on a single appliance, and racing them fails whichever "+
				"loses", rel)
		}
	}
}
