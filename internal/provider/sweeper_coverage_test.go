package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// resourceSweeperExclusions documents every internal/resources/*.go file
// that MUST NOT have a sweeper registered. Every exclusion falls into one
// of three categories:
//
//   - singleton: The resource represents a system-wide PUT-style config
//     (e.g. ftp_config, smb_config). There is no list endpoint to
//     enumerate instances, and sweeping would mean "reset the host
//     configuration", which is catastrophic on any real test system.
//
//   - dangerous: Sweeping would destroy or reconfigure real infrastructure
//     that cannot be distinguished from test fixtures (pool, service,
//     network_interface). A sweeper here is a safety hazard.
//
//   - pending: A genuine listable resource that does not yet have a
//     sweeper function. Each entry is a known gap tracked as tech debt;
//     converting one to a registered sweeper is always welcome.
//
// Any resource file in internal/resources/ that is NOT in the registered
// sweeper set AND NOT in this exclusion map causes TestSweeperCoverage
// to fail. That prevents accidental drift between the sweeper set and
// the resource set as the provider grows.
var resourceSweeperExclusions = map[string]string{
	"alertclasses":      "singleton: Delete resets alertclasses to {} — no list endpoint, no fixtures to sweep",
	"catalog":           "singleton: Delete resets preferred_trains — catalog itself cannot be destroyed",
	"directoryservices": "singleton: Delete resets directory services config to defaults",
	"ftp_config":        "singleton: Delete resets FTP config to defaults",
	"kmip_config":       "singleton: Delete resets KMIP config to defaults",
	"mail_config":       "singleton: Delete resets mail config to defaults",
	"network_config":    "singleton: Delete resets network general config",
	"nfs_config":        "singleton: Delete resets NFS server config to defaults",
	"nvmet_global":      "singleton: Delete resets NVMe-oF global config",
	"smb_config":        "singleton: Delete resets SMB server config to defaults",
	"snmp_config":       "singleton: Delete resets SNMP config to defaults",
	"ssh_config":        "singleton: Delete resets SSH config to defaults",
	"system_update":     "singleton: Delete is a no-op by design (prevents surprise-reboot footgun); no list endpoint, no fixtures to sweep",
	"systemdataset":     "singleton: Delete resets system dataset pool to boot-pool",
	"ups_config":        "singleton: Delete resets UPS config to defaults",
	"network_interface": "dangerous: listing returns real NICs that cannot be distinguished from fixtures — sweeping risks host disconnect",
	"pool":              "dangerous: sweeping pools would destroy storage backing real filesystems",
	"service":           "dangerous: sweeping services would disable running system services",
	"dns_nameserver":    "pending: part of /network/configuration nameservers slice, needs composite sweeper",
	"filesystem_acl":    "pending: not directly listable — path-based; needs a known-fixture-paths inventory",
}

// sweeperRegistrationRE matches every call to resource.AddTestSweepers
// in sweeper_test.go so the coverage test can derive the canonical
// registered-sweeper set from the single source of truth (the init()
// block) rather than maintaining a parallel list.
var sweeperRegistrationRE = regexp.MustCompile(`resource\.AddTestSweepers\("truenas_(\w+)"`)

// TestSweeperCoverage verifies that every resource in internal/resources/
// has either a registered sweeper or an explicit exclusion with rationale.
// This guards against silently dropping sweeper coverage when new resources
// are added and keeps the exclusion list honest by also failing on stale
// entries that reference resources which no longer exist.
func TestSweeperCoverage(t *testing.T) {
	src, err := os.ReadFile("sweeper_test.go")
	if err != nil {
		t.Fatalf("read sweeper_test.go: %v", err)
	}
	registered := map[string]struct{}{}
	for _, m := range sweeperRegistrationRE.FindAllStringSubmatch(string(src), -1) {
		registered[m[1]] = struct{}{}
	}
	if len(registered) == 0 {
		t.Fatal("no registered sweepers parsed from sweeper_test.go — regex broken?")
	}

	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob internal/resources: %v", err)
	}
	resourceSet := map[string]struct{}{}
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		resourceSet[strings.TrimSuffix(base, ".go")] = struct{}{}
	}
	if len(resourceSet) == 0 {
		t.Fatal("no resource files found in ../resources — running from wrong dir?")
	}

	var missing []string
	for name := range resourceSet {
		if _, ok := registered[name]; ok {
			continue
		}
		if _, ok := resourceSweeperExclusions[name]; ok {
			continue
		}
		missing = append(missing, name)
	}
	if len(missing) > 0 {
		t.Fatalf("the following resources have neither a registered sweeper "+
			"nor an entry in resourceSweeperExclusions: %v\n\n"+
			"Every new resource MUST either:\n"+
			"  1. Register a sweeper in internal/provider/sweeper_test.go init(), OR\n"+
			"  2. Be added to resourceSweeperExclusions with a rationale.",
			missing)
	}

	var stale []string
	for name := range resourceSweeperExclusions {
		if _, ok := resourceSet[name]; !ok {
			stale = append(stale, name)
		}
	}
	if len(stale) > 0 {
		t.Fatalf("resourceSweeperExclusions contains entries for resources that "+
			"no longer exist in internal/resources: %v", stale)
	}

	var overlap []string
	for name := range resourceSweeperExclusions {
		if _, ok := registered[name]; ok {
			overlap = append(overlap, name)
		}
	}
	if len(overlap) > 0 {
		t.Fatalf("resources appear in BOTH the exclusion list and the registered "+
			"sweepers: %v — pick one", overlap)
	}
}
