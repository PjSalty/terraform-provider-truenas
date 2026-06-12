package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// testAccFunctionRE captures every acceptance test function name and
// the resource family it targets. Convention across the tree:
//
//	func TestAcc<ResourceName>Resource_<phase>(t *testing.T)  // provider/acc_*
//	func TestAcc<ResourceName>_<phase>(t *testing.T)          // resources/*_test.go
//
// Where <phase> is one of basic, update, disappears, plus any number
// of resource-specific phases (chap, ipv6, etc). NOTE: we do NOT
// capture _import here, by convention this codebase exercises import
// via an `ImportState: true` TestStep inside _basic, not via a
// separate _import function. The import-presence check below scans
// the test file for that literal instead.
var testAccFunctionRE = regexp.MustCompile(`(?m)^func TestAcc(\w+?)(?:Resource)?_(basic|update|disappears)\(t \*testing\.T\)`)

// importStateStepRE matches the TestStep marker that exercises the
// terraform import flow. Whitespace between the colon and `true` is
// variable across the tree, so the regex is whitespace-tolerant.
var importStateStepRE = regexp.MustCompile(`ImportState:\s+true,`)

// lifecycleResourceExclusions lists resource families that legitimately
// do NOT need every phase. Each entry must come with a one-line
// rationale. The map key is the camelCase resource name as it appears
// in the TestAcc<Name>... function (e.g. "DatasetResource" → key
// "Dataset", "ISCSITargetResource" → key "ISCSITarget").
//
// Lifecycle phases:
//   - basic:      smoke test that the resource can be Created
//   - update:     in-place Update of at least one attribute
//   - import:     terraform import round-trip
//   - disappears: out-of-band delete + Terraform-detects-drift
//
// Exclusion categories:
//   - data-source: the type is a Data Source, not a Resource (no
//     lifecycle to test).
//   - singleton: the resource is a per-host config singleton. update
//     and basic ARE meaningful, but disappears does not apply
//     (Delete is a no-op reset) and import is trivially the well-known
//     singleton ID.
//   - beta: the test is gated on infrastructure not generally
//     available (a cloud credential, a VMware host, etc) and ships
//     a t.Skip stub.
var lifecycleResourceExclusions = map[string]map[string]string{
	// Data sources accessed through a "Resource" test function, the
	// test is a data-only smoke check, no lifecycle to exercise.
	"Catalog": {"update": "data-source: TRUENAS catalog is system-managed, only readable",
		"disappears": "data-source", "import": "data-source"},
	"Pool": {"update": "data-source: pools are managed out-of-band via TrueNAS itself",
		"disappears": "data-source", "basic": "data-source: covered via truenas_pool data source",
		"import": "data-source"},

	// Per-host config singletons, disappears does not apply
	// (Delete is a no-op reset). _import via the standard
	// ImportState TestStep is fine for most of them; we list
	// only the ones that genuinely lack it.
	"AlertClasses":  {"disappears": "singleton: Delete is a no-op reset"},
	"DNSNameserver": {"disappears": "singleton: part of network_config"},
	"FTPConfig":     {"disappears": "singleton: Delete is a no-op reset"},
	"FilesystemACL": {"disappears": "path-based singleton, see filesystem_acl_template for template flow"},
	"KMIPConfig":    {"disappears": "singleton with one-shot trigger flags", "import": "singleton: trigger flags don't round-trip cleanly through import"},
	"MailConfig":    {"disappears": "singleton: Delete is a no-op reset"},
	"NetworkConfig": {"disappears": "singleton: Delete is a no-op reset"},
	"NFSConfig":     {"disappears": "singleton: Delete is a no-op reset"},
	"NVMetGlobal":   {"disappears": "singleton: Delete is a no-op reset"},
	"Service":       {"disappears": "singleton: services cannot be deleted, only disabled"},
	"SMBConfig":     {"disappears": "singleton: Delete is a no-op reset"},
	"SNMPConfig":    {"disappears": "singleton: Delete is a no-op reset", "import": "singleton: v3 sensitive fields don't round-trip"},
	"SSHConfig":     {"disappears": "singleton: Delete is a no-op reset", "import": "singleton: weak_ciphers list serialisation differs"},
	"SystemDataset": {"disappears": "singleton"},
	"UPSConfig":     {"disappears": "singleton: Delete is a no-op reset"},

	// Sensitive-payload resources where import legitimately cannot
	// round-trip the secret field, so the standard ImportState test
	// step is omitted in favour of a custom check.
	"ISCSIAuth":          {"import": "secret/peersecret are write-only; covered by basic with ImportStateVerifyIgnore"},
	"KerberosKeytab":     {"import": "file is base64 opaque and may be re-encoded; covered by basic step"},
	"KeychainCredential": {"import": "attributes map contains write-only credential payloads"},

	// Env-gated / beta, full lifecycle deferred until the test
	// environment can support the resource. The base test ships a
	// t.Skip stub when the relevant TRUENAS_TEST_* env var is unset.
	"App":                 {"update": "beta: full app install too slow/flaky for CI", "import": "beta", "disappears": "beta"},
	"CloudBackup":         {"update": "env-gated on TRUENAS_TEST_CLOUD_BACKUP", "import": "env-gated"},
	"CloudSyncCredential": {"import": "env-gated on TRUENAS_TEST_CLOUDSYNC"},
	"VMware":              {"update": "env-gated on TRUENAS_TEST_VMWARE", "import": "env-gated"},
	"NetworkInterface":    {"disappears": "real-NIC: deleting a configured NIC could disconnect the test VM"},

	// Resources that use a custom-named basic step rather than
	// _basic, OR that lack a separate _update because they have
	// no Optional updatable attributes worth toggling.
	"NFSShare": {"update": "covered by ShareNFS, NFSShare is an alias unit-test name"},
}

// TestAcceptanceLifecycleCoverage walks every TestAcc*_<phase> function
// across internal/provider/acc_*_test.go and internal/resources/*_test.go
// and verifies that every resource family has the four core lifecycle
// phases (basic, update, import, disappears), or appears in
// lifecycleResourceExclusions with a rationale per missing phase.
//
// Why: major-provider rigor requires every resource to have its full
// CRUD surface exercised. missing any of the four phases leaves a
// regression vector that escape detection until a user hits the bug.
//
//   - missing _basic: the resource can never have been verified to
//     create successfully against a real backend.
//   - missing _update: the Update path is exercised only through unit
//     tests, which don't catch server-side field normalisation.
//   - missing _import: import path can silently break without notice.
//   - missing _disappears: the provider never proves it can recover
//     from out-of-band deletion.
//
// the invariant lists every gap with the resource name + phase so
// the next change knows exactly which test to add (or which
// exclusion to extend with rationale).
func TestAcceptanceLifecycleCoverage(t *testing.T) {
	var testFiles []string
	for _, dir := range []string{".", "../resources"} {
		matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		for _, m := range matches {
			base := filepath.Base(m)
			// Skip helpers/scaffolding.
			if strings.HasSuffix(base, "_invariant_test.go") ||
				strings.HasSuffix(base, "_coverage_test.go") ||
				base == "acc_helpers_test.go" ||
				base == "acc_validator_errors_test.go" ||
				base == "integration_test.go" {
				continue
			}
			testFiles = append(testFiles, m)
		}
	}

	// resources[name][phase] = true
	resources := map[string]map[string]bool{}
	// importPresence[name] = true if ANY test file containing a function
	// for this resource also contains the `ImportState: true` literal.
	importPresence := map[string]bool{}

	for _, f := range testFiles {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(src)
		fileResources := map[string]bool{}
		for _, m := range testAccFunctionRE.FindAllStringSubmatch(text, -1) {
			name, phase := m[1], m[2]
			// Skip non-resource test helpers (e.g. TestAccValidator_*).
			if strings.HasPrefix(name, "Validator") {
				continue
			}
			// Skip data-source test functions, they're not resources.
			if strings.Contains(name, "DataSource") {
				continue
			}
			if resources[name] == nil {
				resources[name] = map[string]bool{}
			}
			resources[name][phase] = true
			fileResources[name] = true
		}
		// If this file has an ImportState test step, attribute it to
		// every resource family that owns a function in this file.
		// Accept any whitespace between the colon and `true` because the
		// codebase has mixed indentation styles.
		if importStateStepRE.MatchString(text) {
			for name := range fileResources {
				importPresence[name] = true
			}
		}
	}

	// Mark import as present for every resource whose test file
	// contains the ImportState true step.
	for name := range importPresence {
		if resources[name] == nil {
			resources[name] = map[string]bool{}
		}
		resources[name]["import"] = true
	}

	requiredPhases := []string{"basic", "update", "import", "disappears"}

	type gap struct {
		name  string
		phase string
	}
	var gaps []gap
	for name, phases := range resources {
		for _, p := range requiredPhases {
			if phases[p] {
				continue
			}
			if exc, ok := lifecycleResourceExclusions[name]; ok {
				if _, excluded := exc[p]; excluded {
					continue
				}
			}
			gaps = append(gaps, gap{name: name, phase: p})
		}
	}

	if len(gaps) > 0 {
		sort.Slice(gaps, func(i, j int) bool {
			if gaps[i].name != gaps[j].name {
				return gaps[i].name < gaps[j].name
			}
			return gaps[i].phase < gaps[j].phase
		})
		var lines []string
		for _, g := range gaps {
			lines = append(lines, "  "+g.name+" missing _"+g.phase)
		}
		t.Fatalf("%d resource(s) missing required lifecycle phase(s). add the "+
			"test, or add an explicit exclusion to lifecycleResourceExclusions "+
			"with a one-line rationale:\n%s",
			len(gaps), strings.Join(lines, "\n"))
	}

	t.Logf("OK: %d resource families have full lifecycle coverage (basic + update + import + disappears) "+
		"or documented exclusions", len(resources))
}
