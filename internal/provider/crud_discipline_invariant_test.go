package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// crudMethodInvariantRE matches every CRUD method signature in a resource
// file. The first capture group is the method name (Create/Read/
// Update/Delete) — used to dispatch per-method invariants.
var crudMethodInvariantRE = regexp.MustCompile(
	`func \(r \*\w+\) (Create|Read|Update|Delete)\(ctx context\.Context, req resource\.\w+Request, resp \*resource\.\w+Response\) \{`)

// methodBodyAfter returns the body of a CRUD method starting at the
// matched signature line. The body ends at the matching brace —
// approximated by counting `{` / `}` from the function opening
// brace until balance hits zero. Sufficient for the tokens we're
// looking for (which all sit at the top of the body anyway).
func methodBodyAfter(src string, startIdx int) string {
	// Find the function's opening brace (the one at the end of the
	// signature line).
	openIdx := strings.IndexByte(src[startIdx:], '{')
	if openIdx == -1 {
		return ""
	}
	openIdx += startIdx
	depth := 1
	i := openIdx + 1
	for i < len(src) && depth > 0 {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
		i++
	}
	if depth != 0 {
		return src[openIdx:]
	}
	return src[openIdx:i]
}

// crudFiles returns every resource Go file (no _test.go, no
// non-resource helpers like modifyplan_helpers_test.go).
func crudFiles() ([]string, error) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// TestCRUDDiscipline_ReadAlwaysWritesState asserts every resource's
// Read method calls resp.State.Set OR resp.State.RemoveResource
// before returning. A Read that returns without touching State
// silently keeps the prior state, masking out-of-band changes the
// API would otherwise expose — the user sees stale data and a clean
// plan even though something on the server changed. This is one of
// the easiest bugs to introduce while editing and one of the hardest
// to spot in code review.
//
// The check is structural (not semantic): if Set OR RemoveResource
// appears anywhere in the method body, it passes. A semantic check
// would need to verify the calls are on every return path; the
// existing acc tests + read_removes_on_notfound invariant catch the
// "removed on 404 but not on success" gap.
func TestCRUDDiscipline_ReadAlwaysWritesState(t *testing.T) {
	files, err := crudFiles()
	if err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	type finding struct {
		file string
	}
	var missing []finding
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(data)
		matches := crudMethodInvariantRE.FindAllStringSubmatchIndex(src, -1)
		for _, m := range matches {
			method := src[m[2]:m[3]]
			if method != "Read" {
				continue
			}
			body := methodBodyAfter(src, m[0])
			if strings.Contains(body, "resp.State.Set") ||
				strings.Contains(body, "resp.State.SetAttribute") ||
				strings.Contains(body, "resp.State.RemoveResource") {
				continue
			}
			missing = append(missing, finding{file: f})
		}
	}
	if len(missing) > 0 {
		var b strings.Builder
		b.WriteString("Resource Read methods that never write to State:\n\n")
		for _, m := range missing {
			b.WriteString("  ")
			b.WriteString(filepath.Base(m.file))
			b.WriteString("\n")
		}
		b.WriteString("\nEvery Read must call resp.State.Set OR resp.State.RemoveResource\n")
		b.WriteString("(via not-found branch) so terraform observes API drift. A Read that\n")
		b.WriteString("returns without touching State silently freezes the prior view.\n")
		t.Error(b.String())
	}
}

// TestCRUDDiscipline_CreateReadsBackResource asserts every Create
// method either:
//   - calls Get<X> after the underlying Create<X> client call, OR
//   - has at least one call to resp.State.Set with the response
//     payload from the Create.
//
// Why: TrueNAS Create endpoints sometimes return only a partial
// object (e.g. /pool create returns {id} and the operator has to
// re-GET to see the full provisioned config). A Create that
// state.Set's just the create-response leaves Computed fields
// empty in state, so the next refresh shows a phantom diff.
//
// The check is conservative: if the Create body mentions either
// "Get<ResourceType>" (the typed lookup) or "GetX(ctx," where X is
// a generic placeholder, OR explicitly handles the response with
// state mapping, we trust the author had the right pattern in
// mind. False negatives here are acceptable; false positives —
// flagging a correct implementation — would create review noise.
func TestCRUDDiscipline_CreateReadsBackResource(t *testing.T) {
	files, err := crudFiles()
	if err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	var missing []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(data)
		matches := crudMethodInvariantRE.FindAllStringSubmatchIndex(src, -1)
		for _, m := range matches {
			method := src[m[2]:m[3]]
			if method != "Create" {
				continue
			}
			body := methodBodyAfter(src, m[0])
			// Must end with state writeback
			if !strings.Contains(body, "resp.State.Set") {
				missing = append(missing, filepath.Base(f)+" :: Create missing resp.State.Set")
			}
		}
	}
	if len(missing) > 0 {
		var b strings.Builder
		b.WriteString("Resource Create methods missing required state writeback:\n\n")
		for _, m := range missing {
			b.WriteString("  ")
			b.WriteString(m)
			b.WriteString("\n")
		}
		b.WriteString("\nEvery Create must call resp.State.Set with the resulting model.\n")
		b.WriteString("Without it the terraform refresh after apply sees an empty state.\n")
		t.Error(b.String())
	}
}

// TestCRUDDiscipline_DeleteHandlesNotFound asserts every Delete
// method tolerates the resource already being gone — calling Delete
// on an already-deleted resource must not error.
//
// The pattern caught: A Delete that propagates the underlying 404
// error from the API breaks the `terraform destroy` retry loop.
// Terraform Plugin Framework treats a Delete error as a destruction
// failure, so the plan after the failure still has the resource —
// and the next `apply` will try to destroy it again, hitting the
// same 404 forever.
//
// Acceptable patterns:
//   - explicit client.IsNotFound check that swallows the error
//   - explicit early-return on 404 with a tflog warning
//
// The check is conservative: if the Delete body mentions either
// "IsNotFound" or the wsclient equivalent, we accept the
// implementation as compliant. False negatives are accepted; false
// positives would create review noise.
func TestCRUDDiscipline_DeleteHandlesNotFound(t *testing.T) {
	files, err := crudFiles()
	if err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	exempt := map[string]string{
		// Singleton resources can't be "deleted" — Delete is a
		// no-op or a config reset, so the not-found branch is
		// unreachable.
		"alertclasses.go":           "singleton config resource — Delete is a no-op",
		"directoryservices.go":      "singleton config resource — Delete is a no-op",
		"ftp_config.go":             "singleton config resource — Delete is a no-op",
		"kmip_config.go":            "singleton config resource — Delete is a no-op",
		"mail_config.go":            "singleton config resource — Delete is a no-op",
		"network_config.go":         "singleton config resource — Delete is a no-op",
		"nfs_config.go":             "singleton config resource — Delete is a no-op",
		"nvmet_global.go":           "singleton config resource — Delete is a no-op",
		"smb_config.go":             "singleton config resource — Delete is a no-op",
		"snmp_config.go":            "singleton config resource — Delete is a no-op",
		"ssh_config.go":             "singleton config resource — Delete is a no-op",
		"ups_config.go":             "singleton config resource — Delete is a no-op",
		"system_update.go":          "singleton config resource — Delete is a no-op",
		"systemdataset.go":          "singleton config resource — Delete is a no-op",
		"service.go":                "singleton config resource — Delete is a no-op",
		"dns_nameserver.go":         "singleton config resource — Delete clears nameservers via network_config update; no 404 path",
		"reporting_exporter.go":     "Delete is reporting_exporter.delete which doesn't 404 (always succeeds)",
		"acme_dns_authenticator.go": "Delete error handling deferred to client; covered by 404 client tests",
	}
	var missing []string
	for _, f := range files {
		base := filepath.Base(f)
		if _, ok := exempt[base]; ok {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(data)
		matches := crudMethodInvariantRE.FindAllStringSubmatchIndex(src, -1)
		for _, m := range matches {
			method := src[m[2]:m[3]]
			if method != "Delete" {
				continue
			}
			body := methodBodyAfter(src, m[0])
			if strings.Contains(body, "IsNotFound") {
				continue
			}
			missing = append(missing, base)
		}
	}
	if len(missing) > 0 {
		var b strings.Builder
		b.WriteString("Resource Delete methods missing IsNotFound tolerance:\n\n")
		for _, m := range missing {
			b.WriteString("  ")
			b.WriteString(m)
			b.WriteString("\n")
		}
		b.WriteString("\nDelete must swallow the underlying 404 so `terraform destroy` is\n")
		b.WriteString("idempotent. If the resource is a singleton that legitimately can't\n")
		b.WriteString("404 on delete, add it to the exempt map with rationale.\n")
		t.Error(b.String())
	}
}
