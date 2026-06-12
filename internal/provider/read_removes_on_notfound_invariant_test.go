package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// readMethodRE matches the start of a resource's Read method so we
// can scope the not-found check to its body. The signature is
// stable across every resource in the provider:
//
//	func (r *<Type>) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
var readMethodRE = regexp.MustCompile(
	`func \(r \*\w+\) Read\(ctx context\.Context, req resource\.ReadRequest, resp \*resource\.ReadResponse\) \{`)

// notFoundHandlerRE matches the standard "remove from state on 404"
// snippet. We accept either the framework helper or a manual call
// to RemoveResource because both produce the same outcome:
//
//	if wsclient.IsNotFound(err) {
//	    resp.State.RemoveResource(ctx)
//	    return
//	}
//
// The literal "RemoveResource" string is the load-bearing token -
// without it, the resource will not be re-created after an out-of-
// band delete and `terraform plan` will keep producing the same
// "update in-place" diff every run.
const removeResourceTokenS = "resp.State.RemoveResource(ctx)"

// singletonsByDesign lists the resource files whose Read MUST NOT
// remove from state on 404. These are singletons (one per host -
// /api/v2.0/ssh, /api/v2.0/system/general, etc.) where the endpoint
// always returns a config record; a 404 here would indicate a
// platform bug, not an out-of-band delete. They get a hard error
// on 404 instead, which is the correct semantics.
//
// If a new resource is added to this list, a comment in its Read
// method must explain why; the invariant test prints the list as a
// hint when a resource is missing a RemoveResource call so the
// distinction stays visible.
var singletonsByDesign = map[string]bool{
	"alertclasses.go":      true,
	"catalog.go":           true, // SCALE 25.04+ has one catalog ("TRUENAS"); 404 = platform bug
	"directoryservices.go": true,
	"dns_nameserver.go":    true, // entries live inside the network_config singleton; 404 = platform bug
	"ftp_config.go":        true,
	"kmip.go":              true,
	"kmip_config.go":       true, // singleton KMIP server config
	"mail_config.go":       true,
	"network_config.go":    true,
	"nfs_config.go":        true,
	"nvmet_global.go":      true, // singleton NVMe-oF subsystem config
	"smb_config.go":        true,
	"snmp_config.go":       true,
	"ssh_config.go":        true,
	"system_update.go":     true,
	"systemdataset.go":     true,
	"ups_config.go":        true,
}

// TestResourcesRemoveFromStateOnNotFound verifies that every resource
// with a per-ID Read endpoint handles "resource was deleted out of
// band" by calling resp.State.RemoveResource(ctx). Without this, the
// next `terraform plan` after an out-of-band delete keeps producing
// the same update-in-place diff, because the framework still has the
// stale config in state and gets a 200 + decoded-default-zero-value
// back from the upstream's NotFound path (or worse, a panic if the
// resource code does not defend against nil dereferences).
//
// This is the per-resource analogue of the framework-level
// "disappears" acceptance test pattern that major providers use. The
// acceptance form requires a live test VM; this static form runs in
// the unit-test layer and gates every PR.
//
// Singletons (one config record per host, ssh_config, smb_config,
// etc.) are listed in singletonsByDesign and skipped because their
// API surface never legitimately returns 404 for an authenticated
// caller. If you add a new singleton, add its filename to the
// singletonsByDesign map AND write a comment in its Read method
// explaining why a 404 is fatal rather than a state-removal signal.
func TestResourcesRemoveFromStateOnNotFound(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}
	var missing []string
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		if singletonsByDesign[base] {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		text := string(src)

		// No Read method in this file, skip. Some resource files
		// split helpers across multiple .go files in the package;
		// the Read method always lives in the primary file.
		if !readMethodRE.MatchString(text) {
			continue
		}

		// The token check is loose on purpose: the goal is to fail
		// on a refactor that drops the RemoveResource call entirely,
		// not to police where it lives in the Read method body.
		if !strings.Contains(text, removeResourceTokenS) {
			missing = append(missing, base)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("the following resources have a Read method but do not call "+
			"resp.State.RemoveResource(ctx). Operators who delete one of these "+
			"resources out of band (UI, API, kubectl, etc.) will see Terraform "+
			"keep planning the same update-in-place diff forever, because the "+
			"provider cannot tell the framework the resource is gone:\n"+
			"  %s\n\n"+
			"Fix by adding the standard not-found arm in the Read method:\n\n"+
			"  obj, err := r.client.GetX(ctx, id)\n"+
			"  if err != nil {\n"+
			"    if wsclient.IsNotFound(err) {\n"+
			"      resp.State.RemoveResource(ctx)\n"+
			"      return\n"+
			"    }\n"+
			"    resp.Diagnostics.AddError(...)\n"+
			"    return\n"+
			"  }\n\n"+
			"If this resource is a singleton (one config record per host, e.g. "+
			"ssh_config), add its filename to singletonsByDesign in this test "+
			"file with a comment explaining why a 404 is fatal rather than a "+
			"state-removal signal.",
			strings.Join(missing, "\n  "))
	}
}
