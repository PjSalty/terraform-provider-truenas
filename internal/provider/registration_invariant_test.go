package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// resourceConstructorRE matches every line in provider.go that registers
// a resource factory. The shape is:
//
//	resources.NewDatasetResource,
//	resources.NewISCSITargetResource,
//	...
//
// We capture the struct name between `New` and `Resource,` so the test
// can correlate it back to the filename (snake_case form) in
// internal/resources/ and internal/provider/acc_*_test.go.
var resourceConstructorRE = regexp.MustCompile(`resources\.New(\w+)Resource,`)

// camelToSnake converts Go-style CamelCase struct names to the
// snake_case filenames used in internal/resources/. Known acronyms
// are collapsed into all-lowercase (e.g. NVMet → nvmet, ISCSITarget
// → iscsi_target) by a two-pass approach: first convert the CamelCase
// to snake, then collapse runs of single letters.
func camelToSnake(s string) string {
	var out []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Only insert underscore if the previous rune was lowercase
			// OR the next rune is lowercase (handle acronym boundaries).
			prev := rune(s[i-1])
			if prev >= 'a' && prev <= 'z' {
				out = append(out, '_')
			} else if i+1 < len(s) {
				next := rune(s[i+1])
				if next >= 'a' && next <= 'z' {
					out = append(out, '_')
				}
			}
		}
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+32) // to lower
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// TestResourceRegistrationMatchesFilesystem parses provider.go for every
// resources.NewXxxResource, factory reference and asserts that the
// corresponding internal/resources/<snake>.go file exists. This closes
// the "added to Resources() list but never wrote the file" gap — a
// common drive-by refactor hazard in larger providers.
//
// The test does NOT enforce a 1:1 between the provider.go list and
// internal/resources/ because a source file without a constructor
// reference is a valid pattern (deprecated, stub, or factored-out
// helper). It only enforces "every registered factory has a file".
func TestResourceRegistrationMatchesFilesystem(t *testing.T) {
	src, err := os.ReadFile("provider.go")
	if err != nil {
		t.Fatalf("read provider.go: %v", err)
	}
	matches := resourceConstructorRE.FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Fatal("no resources.NewXxxResource references parsed from provider.go — regex broken?")
	}

	// Known tricky mappings that the naive camelToSnake would get wrong.
	// Every acronym-heavy name that does NOT camelToSnake cleanly goes
	// here with its canonical filename.
	specialCases := map[string]string{
		"VMware":                "vmware",
		"VMwareDataSource":      "vmware",
		"KMIPConfig":            "kmip_config",
		"NFSConfig":             "nfs_config",
		"SMBConfig":             "smb_config",
		"SNMPConfig":            "snmp_config",
		"SSHConfig":             "ssh_config",
		"UPSConfig":             "ups_config",
		"FTPConfig":             "ftp_config",
		"DNSNameserver":         "dns_nameserver",
		"ACMEDNSAuthenticator":  "acme_dns_authenticator",
		"APIKey":                "api_key",
		"VM":                    "vm",
		"VMDevice":              "vm_device",
		"ISCSITarget":           "iscsi_target",
		"ISCSIPortal":           "iscsi_portal",
		"ISCSIExtent":           "iscsi_extent",
		"ISCSIInitiator":        "iscsi_initiator",
		"ISCSIAuth":             "iscsi_auth",
		"ISCSITargetExtent":     "iscsi_targetextent",
		"NVMetHost":             "nvmet_host",
		"NVMetSubsys":           "nvmet_subsys",
		"NVMetPort":             "nvmet_port",
		"NVMetNamespace":        "nvmet_namespace",
		"NVMetHostSubsys":       "nvmet_host_subsys",
		"NVMetPortSubsys":       "nvmet_port_subsys",
		"NVMetGlobal":           "nvmet_global",
		"NFSShare":              "share_nfs",
		"SMBShare":              "share_smb",
		"TLSConfig":             "tls_config",
		"SystemDataset":         "systemdataset",
		"DirectoryServices":     "directoryservices",
		"AlertClasses":          "alertclasses",
		"CloudSyncCredential":   "cloudsync_credential",
		"CloudSync":             "cloud_sync",
		"CloudBackup":           "cloud_backup",
		"ReportingExporter":     "reporting_exporter",
		"FilesystemACL":         "filesystem_acl",
		"FilesystemACLTemplate": "filesystem_acl_template",
		"KerberosRealm":         "kerberos_realm",
		"KerberosKeytab":        "kerberos_keytab",
		"KeychainCredential":    "keychain_credential",
		"NetworkInterface":      "network_interface",
		"NetworkConfig":         "network_config",
		"MailConfig":            "mail_config",
		"AlertService":          "alert_service",
		"StaticRoute":           "static_route",
		"SnapshotTask":          "snapshot_task",
		"RsyncTask":             "rsync_task",
		"ScrubTask":             "scrub_task",
		"CronJob":               "cronjob",
		"InitScript":            "init_script",
	}

	var gaps []string
	for _, m := range matches {
		camel := m[1]
		// Skip DataSource constructors — they live in internal/datasources/
		// and this test is about resources/.
		if strings.HasSuffix(camel, "Data") {
			continue
		}
		snake, ok := specialCases[camel]
		if !ok {
			snake = camelToSnake(camel)
		}
		path := filepath.Join("..", "resources", snake+".go")
		if _, err := os.Stat(path); err != nil {
			gaps = append(gaps, camel+" → "+path)
		}
	}
	if len(gaps) > 0 {
		t.Fatalf("the following resources are registered in provider.go but have no "+
			"corresponding source file in internal/resources/:\n\n  %s\n\n"+
			"Either (a) add the file, (b) remove the registration, or (c) if the struct "+
			"name requires an irregular snake_case mapping, add it to specialCases in "+
			"registration_invariant_test.go.", strings.Join(gaps, "\n  "))
	}
}
