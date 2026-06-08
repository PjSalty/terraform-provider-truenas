package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// importStateIgnoreFieldsRE captures every field name passed to
// ImportStateVerifyIgnore across the test tree. The form we accept:
//
//	ImportStateVerifyIgnore: []string{"foo", "bar"},
//	ImportStateVerifyIgnore: []string{
//	    "foo",
//	    "bar",
//	},
//
// The regex is greedy from the opening `{` to the closing `}` and we
// then extract individual quoted strings inside.
var importStateIgnoreRE = regexp.MustCompile(`(?s)ImportStateVerifyIgnore:\s*\[\]string\{([^}]*)\}`)

// fieldRE pulls individual quoted strings out of the captured body.
var ignoreFieldRE = regexp.MustCompile(`"([^"]+)"`)

// allowedIgnoreFields documents EVERY (test-file, field-name) pair
// that legitimately needs to be excluded from terraform import
// round-trip verification. The map key is "<test_file_basename>::<field>"
// and the value is a one-line rationale. The value MUST mention the
// specific reason: secret/sensitive, server-mutated, write-only flag,
// computed-but-not-readable, etc.
//
// If a test adds a new ImportStateVerifyIgnore entry, this map MUST
// gain a matching key + rationale or the invariant fails. The point
// is to prevent the "just add it to the ignore list to make the test
// pass" anti-pattern that silently hides real Read/Create shape bugs.
//
// To audit: read each rationale and ask "is this actually unavoidable
// or did we hide a Read-side bug?" The honest answer determines
// whether the resource needs a fix or whether the field genuinely
// can't round-trip (true secrets, write-only flags, etc.).
var allowedIgnoreFields = map[string]string{
	// Secrets / sensitive write-only inputs that the API never returns.
	"acc_iscsi_auth_test.go::secret":                    "CHAP secret: write-only; TrueNAS returns it masked on read",
	"acc_iscsi_auth_test.go::peersecret":                "CHAP peer secret: write-only; masked on read",
	"iscsi_auth_test.go::secret":                        "CHAP secret: write-only; masked on read",
	"iscsi_auth_test.go::peersecret":                    "CHAP peer secret: write-only; masked on read",
	"acc_api_key_test.go::key":                          "API key value: only returned once at create time, never on read",
	"acc_user_test.go::password":                        "user password: write-only; never returned on read",
	"acc_user_test.go::group_create":                    "convenience flag: tells provider to create primary group; not stored upstream",
	"user_test.go::password":                            "user password: write-only; never returned on read",
	"user_test.go::group_create":                        "convenience flag: tells provider to create primary group; not stored upstream",
	"acc_nvmet_host_test.go::dhchap_key":                "NVMe-oF DH-CHAP secret: write-only; never returned",
	"acc_nvmet_host_test.go::dhchap_ctrl_key":           "NVMe-oF DH-CHAP control secret: write-only; never returned",
	"nvmet_host_test.go::dhchap_key":                    "NVMe-oF DH-CHAP secret: write-only; never returned",
	"nvmet_host_test.go::dhchap_ctrl_key":               "NVMe-oF DH-CHAP control secret: write-only; never returned",
	"snmp_config_test.go::v3_password":                  "SNMPv3 auth password: write-only; never returned",
	"snmp_config_test.go::v3_privpassphrase":            "SNMPv3 priv passphrase: write-only; never returned",
	"cloud_backup_test.go::password":                    "restic repo password: write-only; never returned",
	"vmware_test.go::password":                          "vCenter password: write-only; never returned",
	"ssh_config_test.go::weak_ciphers":                  "list serialised inconsistently between create and read; raised upstream",

	// Sensitive JSON blobs where the API masks values inside.
	"acc_alert_service_test.go::settings_json":          "settings_json contains the alert receiver password which TrueNAS masks on read",
	"alert_service_test.go::settings_json":              "settings_json contains the alert receiver password which TrueNAS masks on read",
	"acc_cloudsync_credential_test.go::provider_attributes_json": "cloud-credential attributes contain S3/B2/etc secret keys masked on read",
	"acc_reporting_exporter_test.go::attributes_json":   "graphite/influx/etc auth tokens inside attributes_json are masked on read",
	"acc_filesystem_acl_template_test.go::acl_json":     "TrueNAS canonicalises ACL JSON server-side; bit-exact round-trip is not guaranteed",

	// Write-only convenience or trigger flags that have no Read counterpart.
	"catalog_test.go::sync_on_create":                   "one-shot trigger: tells provider to run a sync after create; not persisted",
	"kmip_config_test.go::change_server":                "one-shot flag: tells the API to rotate to a new KMIP server",
	"kmip_config_test.go::validate":                     "one-shot flag: tells the API to validate the config",
	"kmip_config_test.go::force_clear":                  "one-shot flag: tells the API to forcibly clear pending operations",
	"directoryservices_test.go::credential_json":        "credential payload is write-only; never returned",
	"directoryservices_test.go::configuration_json":     "configuration payload is normalised server-side; not bit-exact",
	"directoryservices_test.go::force":                  "one-shot flag: tells the API to force-reconfigure",
	"acc_keychain_credential_test.go::attributes":       "credential attributes contain SSH private key / API token data masked on read",
	"acc_vm_device_test.go::attributes":                 "device attributes shape is poly-typed (DISK/NIC/etc); JSON marshal differs from create",
	"acc_vm_device_test.go::vm":                         "vm reference is stored as ID upstream; never round-trips as the input form",
	"acc_kerberos_keytab_test.go::file":                 "keytab is base64 opaque; TrueNAS may re-encode on store, no bit-exact guarantee",
	"kerberos_keytab_test.go::file":                     "keytab is base64 opaque; TrueNAS may re-encode on store, no bit-exact guarantee",

	// Computed/derived fields that are intentionally not round-tripped.
	"vm_test.go::status":                                "computed status reflects runtime state, not config",

	// App / container chart values: TrueNAS canonicalises and may strip nulls.
	"app_test.go::values":                               "chart values JSON is canonicalised by TrueNAS; not bit-exact on read",
	"app_test.go::catalog_app":                          "create-time selector: replaced by the resolved chart ID on read",
	"app_test.go::train":                                "create-time selector: replaced by the resolved chart ID on read",
	"app_test.go::remove_images":                        "destroy-time flag: not persisted",
	"app_test.go::remove_ix_volumes":                    "destroy-time flag: not persisted",

	// Storage edge cases.
	"dataset_test.go::parent_dataset":                   "convenience attribute used to scope ZFS path resolution; not stored",
	"zvol_test.go::volsize":                             "TrueNAS rounds volsize to the nearest volblocksize multiple; the read value can differ from the requested input bytes",
	"zvol_test.go::volblocksize":                        "set at create-time only; treated as immutable RequiresReplace and not round-tripped via state import on every test fixture",

	// Certificate create-flow inputs that don't round-trip.
	"acc_certificate_test.go::privatekey":               "private key is write-only at import; only the certificate body is returned on read",
	"acc_certificate_test.go::create_type":              "create_type is a create-time selector (CERTIFICATE_CREATE_INTERNAL etc); not stored upstream",
}

// TestImportStateVerifyIgnoreEntriesAreDocumented verifies that every
// ImportStateVerifyIgnore field passed in any test file is listed in
// the allowedIgnoreFields registry above with a rationale.
//
// Why: ImportStateVerifyIgnore is the single biggest hiding place for
// Read/Create shape bugs in a Terraform provider. The "just add it to
// the ignore list" reflex covers up real round-trip bugs where the
// Read path doesn't reconstruct what Create posted. Forcing every
// entry through a registry with rationale turns adding one into a
// deliberate act that has to survive review.
//
// To extend: when a test legitimately needs a new ignore, add a
// "<file_basename>::<field>" entry to allowedIgnoreFields with the
// shortest honest reason. If the reason is "Read doesn't return
// what we Create" — that's a provider bug, fix it instead of adding
// to the registry.
func TestImportStateVerifyIgnoreEntriesAreDocumented(t *testing.T) {
	var testFiles []string
	for _, dir := range []string{"../resources", "."} {
		ms, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		testFiles = append(testFiles, ms...)
	}

	type entry struct {
		file  string
		field string
	}
	var seen []entry

	for _, f := range testFiles {
		// Skip self — the invariant test contains regex examples that
		// look like real ImportStateVerifyIgnore entries to the scanner.
		if strings.HasSuffix(f, "importstate_verify_ignore_invariant_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, m := range importStateIgnoreRE.FindAllStringSubmatch(string(src), -1) {
			body := m[1]
			for _, fm := range ignoreFieldRE.FindAllStringSubmatch(body, -1) {
				seen = append(seen, entry{
					file:  filepath.Base(f),
					field: fm[1],
				})
			}
		}
	}

	if len(seen) == 0 {
		t.Fatal("no ImportStateVerifyIgnore entries found — regex broken?")
	}

	// Find entries missing from the registry.
	var undocumented []entry
	for _, e := range seen {
		key := e.file + "::" + e.field
		if _, ok := allowedIgnoreFields[key]; !ok {
			undocumented = append(undocumented, e)
		}
	}

	if len(undocumented) > 0 {
		sort.Slice(undocumented, func(i, j int) bool {
			if undocumented[i].file != undocumented[j].file {
				return undocumented[i].file < undocumented[j].file
			}
			return undocumented[i].field < undocumented[j].field
		})
		var lines []string
		for _, e := range undocumented {
			lines = append(lines, "  \""+e.file+"::"+e.field+"\": \"<rationale>\",")
		}
		t.Fatalf("%d undocumented ImportStateVerifyIgnore entries — add each to "+
			"allowedIgnoreFields with a one-line rationale. If the reason is \"Read "+
			"doesn't return what Create posted\" that is a provider bug, fix it "+
			"instead of hiding it:\n%s",
			len(undocumented), strings.Join(lines, "\n"))
	}

	// Find stale entries in the registry that don't match any test.
	seenSet := map[string]struct{}{}
	for _, e := range seen {
		seenSet[e.file+"::"+e.field] = struct{}{}
	}
	var stale []string
	for key := range allowedIgnoreFields {
		if _, ok := seenSet[key]; !ok {
			stale = append(stale, key)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf("%d stale entries in allowedIgnoreFields that reference test/field pairs "+
			"that no longer exist in the tree — remove them to keep the registry honest:\n  %s",
			len(stale), strings.Join(stale, "\n  "))
	}

	t.Logf("OK: %d documented ImportStateVerifyIgnore entries across the test tree", len(seen))
}
