package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// schemaVersionRE matches the `Version: <N>` line inside a Schema()
// implementation. The plugin-framework convention is:
//
//	func (r *MyResource) Schema(...) {
//	    resp.Schema = schema.Schema{
//	        Version:     1,
//	        ...
//
// Any non-zero version is a real schema migration that breaks state
// compatibility for users who pinned to an older provider version —
// unless the resource ships a corresponding StateUpgrader and a test
// that proves the upgrade works.
var schemaVersionRE = regexp.MustCompile(`(?m)^\s+Version:\s+(\d+),`)

// upgradeStateInterfaceRE matches the framework interface-satisfaction
// line that resources include when they implement schema upgrades:
//
//	var _ resource.ResourceWithUpgradeState = &MyResource{}
//
// We use this rather than looking for the UpgradeState method body
// because the interface line is what Terraform actually dispatches on,
// and it's a stable single-line declaration that's easy to grep.
var upgradeStateInterfaceRE = regexp.MustCompile(`resource\.ResourceWithUpgradeState`)

// TestResourcesWithSchemaVersionHaveUpgradeState verifies the migration
// contract: any resource that ships `Version: N` (N > 0) in its schema
// MUST also satisfy the ResourceWithUpgradeState interface AND must
// have a corresponding `*_upgradestate_test.go` covering the migration.
//
// Why this matters: bumping a schema version is a state-format break.
// Existing users who run `terraform apply` against a new provider
// version will have their state silently mis-parsed unless the
// resource implements StateUpgrader for every prior version. Forgetting
// to ship the upgrader is one of the highest-blast-radius mistakes a
// provider author can make — it manifests as either "all attributes
// nulled out" or "plan wants to recreate everything" on the user's
// next plan, which can mean data loss.
//
// The test scans every internal/resources/*.go that defines a
// non-test struct, parses out the schema Version, and fails if the
// version is > 0 but either the interface assertion is missing or
// the *_upgradestate_test.go file is absent.
func TestResourcesWithSchemaVersionHaveUpgradeState(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}

	type finding struct {
		file       string
		version    string
		hasIface   bool
		testExists bool
	}
	var findings []finding

	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		s := string(src)

		// Find the schema version. Multiple matches are possible if a
		// resource has nested schemas — we only care about the first
		// top-level Version: line.
		vmatch := schemaVersionRE.FindStringSubmatch(s)
		if vmatch == nil {
			continue
		}
		version := vmatch[1]
		if version == "0" {
			// Version 0 is the implicit default; no migration needed.
			continue
		}

		// Look for the interface assertion.
		hasIface := upgradeStateInterfaceRE.MatchString(s)

		// Look for a sibling *_upgradestate_test.go file.
		base := strings.TrimSuffix(filepath.Base(m), ".go")
		testFile := filepath.Join(filepath.Dir(m), base+"_upgradestate_test.go")
		_, err = os.Stat(testFile)
		testExists := err == nil

		findings = append(findings, finding{
			file:       m,
			version:    version,
			hasIface:   hasIface,
			testExists: testExists,
		})
	}

	if len(findings) == 0 {
		t.Log("no resources with Version > 0 found — invariant has nothing to enforce yet")
		return
	}

	var failed bool
	for _, f := range findings {
		base := strings.TrimSuffix(filepath.Base(f.file), ".go")
		if !f.hasIface {
			t.Errorf("%s ships Version: %s but does not implement ResourceWithUpgradeState — "+
				"existing users' state will silently break on upgrade. Either revert the version "+
				"bump or add the interface and a StateUpgrader.",
				f.file, f.version)
			failed = true
		}
		if !f.testExists {
			t.Errorf("%s ships Version: %s but no %s_upgradestate_test.go exists — "+
				"schema migrations MUST have a test that round-trips a prior-version state "+
				"through the upgrader and asserts the result. Otherwise the migration is "+
				"un-validated and the next refactor breaks it without warning.",
				f.file, f.version, base)
			failed = true
		}
		if f.hasIface && f.testExists {
			t.Logf("OK  %s Version: %s + upgradestate test", base, f.version)
		}
	}

	if failed {
		t.Log("if you intentionally bumped Version without an upgrader, that is a state-format " +
			"breaking change that will lose data for existing users — almost never the right call. " +
			"revert the version bump and use a different field name, or add the full migration.")
	}
}
