package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// dataSourceTestFileRE matches data-source acceptance test filenames.
// Data sources have no destroy path (Terraform doesn't manage their
// lifecycle), so CheckDestroy is structurally meaningless.
var dataSourceTestFileRE = regexp.MustCompile(`_data_test\.go$`)

// singletonResourceTestFiles lists test filenames whose resource is
// a per-host singleton (one config record API-side). For these,
// "destroy" maps to "reset the singleton's fields to their defaults"
// rather than "remove the row from the upstream." A CheckDestroy
// callback CAN still be meaningful (verify defaults are restored)
// but is not load-bearing for state-correctness, so we treat it as
// optional with a per-file note.
//
// Each entry should mirror an entry in singletonsByDesign (see
// read_removes_on_notfound_invariant_test.go); the two lists are
// allowed to diverge intentionally — see the comment on each.
var singletonResourceTestFiles = map[string]string{
	"alertclasses_test.go":      "singleton — destroy resets the singleton alertclasses config to default",
	"directoryservices_test.go": "singleton — destroy disables the directory service",
	"dns_nameserver_test.go":    "config entries live inside the network_config singleton",
	"filesystem_acl_test.go":    "path-scoped — destroy resets ACL to defaults on the path, the path itself is the dataset's concern",
	"ftp_config_test.go":        "singleton — destroy resets the singleton",
	"kmip_config_test.go":       "singleton — destroy resets the singleton",
	"mail_config_test.go":       "singleton — destroy resets the singleton",
	"network_config_test.go":    "singleton — destroy resets the singleton",
	"nfs_config_test.go":        "singleton — destroy resets the singleton",
	"smb_config_test.go":        "singleton — destroy resets the singleton",
	"snmp_config_test.go":       "singleton — destroy resets the singleton",
	"ssh_config_test.go":        "singleton — destroy resets the singleton",
	"systemdataset_test.go":     "singleton — destroy resets pool assignment to boot pool",
	"system_update_test.go":     "singleton — destroy resets the singleton",
	"ups_config_test.go":        "singleton — destroy resets the singleton",
}

// TestAcceptanceTestsHaveCheckDestroy verifies that every TestAcc
// function for a real resource (not a data source, not a singleton)
// registers a CheckDestroy callback in its resource.TestCase. Without
// CheckDestroy, the destroy step is exercised by the framework but
// the framework can't verify the upstream actually removed the
// resource — a leaking destroy code path produces a green test on
// state that's silently broken server-side.
//
// This is the per-acc-test analogue of major-provider rigor
// (terraform-provider-aws explicitly fails its CI when a TestAcc is
// missing CheckDestroy). The static-analysis form here runs in the
// unit-test layer with no live TrueNAS required, so it gates every
// PR alongside the existing 100% coverage rule.
//
// Allowlist:
//   - *_data_test.go: data source tests have no destroy lifecycle.
//   - singletonResourceTestFiles: destroy = reset to default, not
//     remove from the upstream. A CheckDestroy that verifies the
//     reset is still desirable; flagging it as MUST would force
//     boilerplate that doesn't catch real bugs.
//   - Tests whose entire body is `t.Skip(...)`: no Steps to verify.
//
// The static-analysis form here is intentionally conservative — the
// allowlist explains every exception so the next contributor can
// see at a glance which tests are off the hook and why.
func TestAcceptanceTestsHaveCheckDestroy(t *testing.T) {
	roots := []string{"../resources", "../datasources"}
	var missing []string
	for _, root := range roots {
		matches, err := filepath.Glob(root + "/*_test.go")
		if err != nil {
			t.Fatalf("glob %s: %v", root, err)
		}
		for _, m := range matches {
			base := filepath.Base(m)
			if dataSourceTestFileRE.MatchString(base) {
				continue
			}
			if _, ok := singletonResourceTestFiles[base]; ok {
				continue
			}
			src, err := os.ReadFile(m)
			if err != nil {
				t.Fatalf("read %s: %v", m, err)
			}
			text := string(src)
			for _, fn := range testAccFuncRE.FindAllStringSubmatch(text, -1) {
				name := fn[1]
				body := extractFunctionBody(text, fn[0])
				if body == "" {
					continue
				}
				// Skip-stub tests have no Steps so CheckDestroy is
				// structurally meaningless.
				if strings.Contains(body, "t.Skip(") &&
					!strings.Contains(body, "resource.Test(") {
					continue
				}
				// Plan-only tests don't apply or destroy anything, so
				// there's nothing for CheckDestroy to verify. The
				// PlanOnly:true + ExpectNonEmptyPlan:true pattern is
				// the standard idiom for "schema validation only" tests
				// that run without a live infrastructure dependency.
				if strings.Contains(body, "PlanOnly:") {
					continue
				}
				if !strings.Contains(body, "CheckDestroy:") {
					missing = append(missing, base+":"+name)
				}
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("the following acceptance tests do not register a CheckDestroy "+
			"callback. Without it, a leaking destroy code path produces a "+
			"green test even when the upstream did not actually remove the "+
			"resource — exactly the kind of silent acceptance-test failure "+
			"this static check is designed to surface:\n  %s\n\n"+
			"Fix by adding the standard pattern to each test:\n\n"+
			"  resource.Test(t, resource.TestCase{\n"+
			"    PreCheck:                 func() { testAccPreCheck(t) },\n"+
			"    ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,\n"+
			"    CheckDestroy:             testAccCheck<Type>Destroy(resourceName),\n"+
			"    Steps: []resource.TestStep{...},\n"+
			"  })\n\n"+
			"The CheckDestroy callback should query the upstream and confirm "+
			"the resource is gone (not just that it has been removed from "+
			"state). See testAccCheckDatasetDestroy in dataset_test.go for "+
			"the reference implementation.\n\n"+
			"If the test exists for a singleton resource (delete = reset to "+
			"default, not actual removal), add its filename to "+
			"singletonResourceTestFiles in this invariant with a one-line "+
			"comment explaining why CheckDestroy is optional for it.",
			strings.Join(missing, "\n  "))
	}
}
