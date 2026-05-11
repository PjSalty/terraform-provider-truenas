package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// deleteNotFoundExclusions lists every resource whose Delete function
// legitimately does NOT call client.IsNotFound. The only valid reason is
// "singleton resource" — a resource that maps to a system-wide PUT-style
// config (ftp_config, smb_config, etc.). For those, Delete resets to a
// default via Update, so there is no DeleteXxx call to guard with
// IsNotFound, and a Delete-while-already-gone scenario is impossible.
//
// This map is exactly the singleton subset of resourceSweeperExclusions
// (see sweeper_coverage_test.go). The two are kept separate because a
// resource CAN be singleton for sweeper purposes but still call
// client.IsNotFound somewhere (e.g. for a nested listing), so the sets
// are not strictly equal in general. They just happen to overlap today.
var deleteNotFoundExclusions = map[string]string{
	"alertclasses":      "singleton: Delete resets via Update, no per-instance DELETE call",
	"catalog":           "singleton: Delete resets preferred_trains via Update",
	"directoryservices": "singleton: Delete resets directory services via Update",
	"dns_nameserver":    "singleton: Delete clears nameservers via UpdateNetworkConfig, no DELETE call",
	"ftp_config":        "singleton: Delete resets FTP config via Update",
	"kmip_config":       "singleton: Delete resets KMIP config via Update",
	"mail_config":       "singleton: Delete resets mail config via Update",
	"network_config":    "singleton: Delete resets network general config via Update",
	"nfs_config":        "singleton: Delete resets NFS server config via Update",
	"nvmet_global":      "singleton: Delete resets NVMe-oF global config via Update",
	"smb_config":        "singleton: Delete resets SMB config via Update",
	"snmp_config":       "singleton: Delete resets SNMP config via Update",
	"ssh_config":        "singleton: Delete resets SSH config via Update",
	"system_update":     "singleton: Delete is a no-op by design (prevents surprise-reboot footgun from accidental destroy)",
	"systemdataset":     "singleton: Delete resets system dataset pool via Update",
	"ups_config":        "singleton: Delete resets UPS config via Update",
}

// deleteFuncRE matches the opening line of any resource's Delete method.
// The Plugin Framework convention we follow is:
//
//	func (r *XxxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
var deleteFuncRE = regexp.MustCompile(`func \([a-z] \*(\w+)Resource\) Delete\(`)

// TestDeleteHandlesNotFound verifies that every non-singleton resource's
// Delete function contains `client.IsNotFound(` (or `IsNotFound(`) in its
// body. The Plugin Framework contract is that Delete on a resource that
// no longer exists on the target system MUST be treated as success — the
// state should just be removed. Without IsNotFound handling, any race
// where the resource is deleted out-of-band between plan and apply turns
// into a confusing, Terraform-fatal error.
//
// The check is a regex over source text rather than an AST walk to keep
// the test simple and fast. It will catch deletions of IsNotFound calls
// even under aggressive refactors, which is all we need for defence.
func TestDeleteHandlesNotFound(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	var violations []string
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		name := strings.TrimSuffix(base, ".go")
		if _, ok := deleteNotFoundExclusions[name]; ok {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		if !deleteFuncRE.Match(src) {
			// No Delete function at all in this file — unusual, but we
			// only care about files that DO define Delete, so skip.
			continue
		}
		if !strings.Contains(string(src), "IsNotFound(") {
			violations = append(violations, name)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("the following resources define Delete but do not call IsNotFound — "+
			"a delete-while-already-gone race will surface as a fatal error instead "+
			"of a graceful state removal: %v\n\n"+
			"Fix by wrapping the client Delete call error check with client.IsNotFound "+
			"(see internal/resources/dataset.go Delete for the reference pattern), "+
			"OR if this resource is a singleton, add it to deleteNotFoundExclusions "+
			"with a rationale.", violations)
	}

	var stale []string
	fileSet := map[string]struct{}{}
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		fileSet[strings.TrimSuffix(base, ".go")] = struct{}{}
	}
	for name := range deleteNotFoundExclusions {
		if _, ok := fileSet[name]; !ok {
			stale = append(stale, name)
		}
	}
	if len(stale) > 0 {
		t.Fatalf("deleteNotFoundExclusions references resources that no longer "+
			"exist in internal/resources: %v", stale)
	}
}
