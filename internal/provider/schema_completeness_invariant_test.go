package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// attributeBlockWithTypeRE is a richer form of attributeBlockRE (which
// captures only name + body), adds a middle capture group for the
// Attribute go-type so the schema-completeness tests can distinguish
// BoolAttribute from StringAttribute when applying exemption rules.
var attributeBlockWithTypeRE = regexp.MustCompile(`(?m)^\s*"([a-z][a-z0-9_]*)":\s*schema\.(\w+Attribute)\{((?:[^{}]|\{[^{}]*\})*?)\}`)

// allowedAttributesWithoutDescription is the LEGACY-ALLOW set:
// attributes that ship today without a Description but predate this
// invariant. Each entry locks a known gap so the invariant stays
// green for the existing codebase while still catching any NEW
// undescribed attribute on the next PR.
//
// The right fix for every entry below is to add a Description.
// Tracked as v2.x polish, backlog item per resource.
//
// Key format: "<resources|datasources>/<file>::<attribute>"
var allowedAttributesWithoutDescription = map[string]string{
	// id attributes, boilerplate "Identifier" description would
	// add value to registry docs; add when touching each file.
	"resources/alertclasses.go::id":            "polish-backlog: id attr lacks Description",
	"resources/cloud_backup.go::id":            "polish-backlog: id attr lacks Description",
	"resources/cronjob.go::id":                 "polish-backlog: id attr lacks Description",
	"resources/filesystem_acl_template.go::id": "polish-backlog: id attr lacks Description",
	"resources/iscsi_auth.go::id":              "polish-backlog: id attr lacks Description",
	"resources/kmip_config.go::id":             "polish-backlog: id attr lacks Description",
	"resources/reporting_exporter.go::id":      "polish-backlog: id attr lacks Description",

	// cronjob, composite schedule fields and CRUD-flag fields lack
	// descriptions across the board. Bulk-add when next touching the
	// file.
	"resources/cronjob.go::command":         "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::description":     "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::enabled":         "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::schedule_dom":    "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::schedule_dow":    "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::schedule_hour":   "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::schedule_minute": "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::schedule_month":  "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::stderr":          "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::stdout":          "polish-backlog: cronjob attr lacks Description",
	"resources/cronjob.go::user":            "polish-backlog: cronjob attr lacks Description",

	// cloud_backup, schedule_* fields share the cronjob pattern;
	// add descriptions when next touching the file.
	"resources/cloud_backup.go::schedule_dom":    "polish-backlog: cloud_backup schedule_* lacks Description",
	"resources/cloud_backup.go::schedule_dow":    "polish-backlog: cloud_backup schedule_* lacks Description",
	"resources/cloud_backup.go::schedule_hour":   "polish-backlog: cloud_backup schedule_* lacks Description",
	"resources/cloud_backup.go::schedule_minute": "polish-backlog: cloud_backup schedule_* lacks Description",
	"resources/cloud_backup.go::schedule_month":  "polish-backlog: cloud_backup schedule_* lacks Description",
}

// TestSchemaCompleteness_EveryAttributeHasDescription scans every
// resource and datasource file for schema attribute declarations and
// fails if any block omits `Description: "..."`.
//
// Why this matters at major-provider rigor:
//
//   - The generated registry docs are built from the Description
//     string. A missing Description produces a blank cell in the
//     attribute table, looks broken to a user evaluating the
//     provider.
//   - `terraform plan` surfaces Description in the attribute
//     diff header. A missing description means the operator has
//     no inline explanation of what the field controls.
//   - tflog traces include Description on field-level events for
//     debugging.
//
// A provider that ships 63 resources can easily grow a dozen
// undescribed attributes over a year of churn. This invariant
// catches it on PR rather than in registry review.
func TestSchemaCompleteness_EveryAttributeHasDescription(t *testing.T) {
	files, err := schemaFiles()
	if err != nil {
		t.Fatalf("enumerate schema files: %v", err)
	}

	type finding struct {
		file string
		attr string
		typ  string
	}
	var missing []finding

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		base := labelForFile(f)
		matches := attributeBlockWithTypeRE.FindAllSubmatch(data, -1)
		for _, m := range matches {
			attr := string(m[1])
			typ := string(m[2])
			body := string(m[3])
			// "Description:" must appear in the block body. The string
			// contents themselves are not inspected, empty descriptions
			// are checked separately by a registry lint we don't
			// duplicate here.
			if strings.Contains(body, "Description:") || strings.Contains(body, "MarkdownDescription:") {
				continue
			}
			key := base + "::" + attr
			if _, allowed := allowedAttributesWithoutDescription[key]; allowed {
				continue
			}
			missing = append(missing, finding{file: f, attr: attr, typ: typ})
		}
	}

	if len(missing) > 0 {
		var b strings.Builder
		b.WriteString("Schema attributes missing Description:\n\n")
		for _, m := range missing {
			b.WriteString("  ")
			b.WriteString(labelForFile(m.file))
			b.WriteString(" : \"")
			b.WriteString(m.attr)
			b.WriteString("\" (")
			b.WriteString(m.typ)
			b.WriteString(")\n")
		}
		b.WriteString("\nEvery public attribute must carry a Description so the registry docs and\n")
		b.WriteString("`terraform plan` output explain the field. Add a description, or, if the\n")
		b.WriteString("attribute is legitimately internal, add an entry to\n")
		b.WriteString("allowedAttributesWithoutDescription with a one-line rationale.\n")
		t.Error(b.String())
	}
}

// TestSchemaCompleteness_RequiredAttributesHaveValidators asserts that
// every attribute declared `Required: true` either carries a
// `Validators:` slice OR is one of the bool/int variants whose
// type system already constrains the value space.
//
// Bug pattern caught: a Required string attribute with no validator
// accepts any input, including empty strings, whitespace-only values,
// or values containing control characters that TrueNAS will reject at
// apply time with a generic "validation failed" surface. Catching
// at plan time is materially better UX.
//
// Exceptions live in allowedRequiredAttributesWithoutValidator with
// a rationale (typically "validation enforced server-side and the
// failure mode is acceptable").
func TestSchemaCompleteness_RequiredAttributesHaveValidators(t *testing.T) {
	files, err := schemaFiles()
	if err != nil {
		t.Fatalf("enumerate schema files: %v", err)
	}

	exemptTypes := map[string]bool{
		"BoolAttribute":   true, // bool is binary; no further validation possible
		"NumberAttribute": true, // arbitrary-precision; rare and not gated
	}

	type finding struct {
		file string
		attr string
		typ  string
	}
	var missing []finding

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		base := labelForFile(f)
		matches := attributeBlockWithTypeRE.FindAllSubmatch(data, -1)
		for _, m := range matches {
			attr := string(m[1])
			typ := string(m[2])
			body := string(m[3])
			if !strings.Contains(body, "Required:    true") &&
				!strings.Contains(body, "Required: true") {
				continue
			}
			if exemptTypes[typ] {
				continue
			}
			if strings.Contains(body, "Validators:") {
				continue
			}
			key := base + "::" + attr
			if _, allowed := allowedRequiredAttributesWithoutValidator[key]; allowed {
				continue
			}
			missing = append(missing, finding{file: f, attr: attr, typ: typ})
		}
	}

	if len(missing) > 0 {
		var b strings.Builder
		b.WriteString("Required attributes without Validators:\n\n")
		for _, m := range missing {
			b.WriteString("  ")
			b.WriteString(labelForFile(m.file))
			b.WriteString(" : \"")
			b.WriteString(m.attr)
			b.WriteString("\" (")
			b.WriteString(m.typ)
			b.WriteString(")\n")
		}
		b.WriteString("\nA Required attribute with no validator accepts any input, including\n")
		b.WriteString("empty strings and whitespace, deferring validation to the TrueNAS\n")
		b.WriteString("API surface which usually returns a generic error message. Add at\n")
		b.WriteString("least a stringvalidator.LengthAtLeast(1) or a typed validator, or\n")
		b.WriteString("document the exception in allowedRequiredAttributesWithoutValidator.\n")
		t.Error(b.String())
	}
}

// allowedRequiredAttributesWithoutValidator lists Required attributes
// that ship without an inline Validators slice. Categorised:
//
//   - data source lookup-by-id / lookup-by-name fields: TrueNAS' API
//     accepts effectively any string for an ID lookup and surfaces a
//     clean 404 if not found. Adding LengthAtLeast(1) would help, but
//     wouldn't catch the more common typo case ("forgotten an entire
//     dataset segment") that only the live API can detect.
//
//   - JSON-blob attributes (settings_json, acl_json, attributes_json):
//     content validation requires parsing the JSON and walking the
//     shape against TrueNAS' attribute schema, which is server-
//     authoritative and version-dependent. Best done at apply time.
//
//   - MapAttribute / ListAttribute with element constraints: the
//     library validators we use don't compose cleanly with nested
//     element-type validators. Schema enforces the type but not
//     per-element shape.
//
// All entries below predate this invariant. The right fix for many
// of them is to add at least a stringvalidator.LengthAtLeast(1).
// Tracked as v2.x polish.
var allowedRequiredAttributesWithoutValidator = map[string]string{
	// data sources, server-side 404 is the source of truth for
	// "this lookup key doesn't exist"; client-side length check
	// would only catch the empty-string typo class.
	"datasources/alert_service.go::id":       "ds-lookup: server-side 404 is source of truth",
	"datasources/api_key.go::id":             "ds-lookup: server-side 404 is source of truth",
	"datasources/app.go::id":                 "ds-lookup: server-side 404 is source of truth",
	"datasources/certificate.go::name":       "ds-lookup: server-side 404 is source of truth",
	"datasources/cronjob.go::id":             "ds-lookup: server-side 404 is source of truth",
	"datasources/dataset.go::id":             "ds-lookup: server-side 404 is source of truth",
	"datasources/disk.go::name":              "ds-lookup: server-side 404 is source of truth",
	"datasources/group.go::name":             "ds-lookup: server-side 404 is source of truth",
	"datasources/iscsi_extent.go::id":        "ds-lookup: server-side 404 is source of truth",
	"datasources/iscsi_initiator.go::id":     "ds-lookup: server-side 404 is source of truth",
	"datasources/iscsi_portal.go::id":        "ds-lookup: server-side 404 is source of truth",
	"datasources/iscsi_target.go::id":        "ds-lookup: server-side 404 is source of truth",
	"datasources/kerberos_realm.go::id":      "ds-lookup: server-side 404 is source of truth",
	"datasources/keychain_credential.go::id": "ds-lookup: server-side 404 is source of truth",
	"datasources/network_interface.go::id":   "ds-lookup: server-side 404 is source of truth",
	"datasources/privilege.go::id":           "ds-lookup: server-side 404 is source of truth",
	"datasources/service.go::service":        "ds-lookup: server-side 404 is source of truth",
	"datasources/share_nfs.go::id":           "ds-lookup: server-side 404 is source of truth",
	"datasources/share_smb.go::id":           "ds-lookup: server-side 404 is source of truth",
	"datasources/snapshot_task.go::id":       "ds-lookup: server-side 404 is source of truth",
	"datasources/user.go::username":          "ds-lookup: server-side 404 is source of truth",
	"datasources/vm.go::id":                  "ds-lookup: server-side 404 is source of truth",

	// JSON-blob attributes, content validation is server-
	// authoritative and version-dependent; a client-side validator
	// would either be a re-implementation of TrueNAS schemas or a
	// shallow length check.
	"resources/cloud_backup.go::attributes_json":     "json-blob: server-authoritative shape",
	"resources/filesystem_acl_template.go::acl_json": "json-blob: server-authoritative shape",
	"resources/cloud_backup.go::password":            "polish-backlog: add LengthAtLeast(1)",
	"resources/keychain_credential.go::attributes":   "map-attr: per-element validation not composed",
	"resources/replication.go::source_datasets":      "polish-backlog: add SizeAtLeast(1)",
	"resources/vm_device.go::attributes":             "map-attr: per-element validation not composed",
}

// schemaFiles returns every resource + datasource Go file (excluding
// _test.go). Used by every schema invariant in this file.
//
// Output paths include the parent directory ("resources/foo.go") so
// findings from a resource and its sister datasource (which often
// share the same basename) don't collide in the error message.
func schemaFiles() ([]string, error) {
	var out []string
	for _, dir := range []string{"../resources", "../datasources"} {
		matches, err := filepath.Glob(dir + "/*.go")
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if strings.HasSuffix(m, "_test.go") {
				continue
			}
			out = append(out, m)
		}
	}
	return out, nil
}

// labelForFile returns a "resources/foo.go" or "datasources/foo.go"
// style label so resource/datasource sister files with the same
// basename are distinguishable in violation messages.
func labelForFile(p string) string {
	dir := filepath.Base(filepath.Dir(p))
	return dir + "/" + filepath.Base(p)
}
