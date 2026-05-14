package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sensitiveAttrBlockRE matches every schema attribute declaration in a
// resource file whose name strongly implies the value is secret. The
// capture groups are (1) the attribute name and (2) the block body
// containing all the schema flags (Sensitive, Optional, etc.).
//
// The body-match regex handles one level of brace nesting, which is
// sufficient for PlanModifiers / Validators slices nested inside an
// Attribute. Matches the same pattern used by
// requires_replace_invariant_test.go.
var sensitiveAttrBlockRE = regexp.MustCompile(`(?m)^\s*"(password|secret|peersecret|api_key|private_?key|privatekey|dhchap_key|dhchap_ctrl_key|client_secret|smtp_password|webhook_secret|webhook_token|access_key|access_token|refresh_token|bearer_token|v3_password|v3_privpassphrase|passphrase|encryption_key|user_secret)":\s*schema\.\w+Attribute\{((?:[^{}]|\{[^{}]*\})*?)\}`)

// allowedNonSensitiveFields documents attribute names matching the
// sensitive-name regex that are LEGITIMATELY non-sensitive. The map
// key is "<resource_file>::<field>" and the value is the rationale.
//
// Today this map is empty. The structure is here for the moment a
// future schema needs to legitimately ship a field with a sensitive-
// looking name as plaintext — for example, a field called "passphrase"
// that holds a public-passphrase fingerprint rather than the secret
// itself. Add the entry with the honest reason at that point.
var allowedNonSensitiveFields = map[string]string{}

// TestSensitiveFieldsAreMarkedSensitive verifies that every schema
// attribute whose name strongly implies the value is a secret carries
// `Sensitive: true` in its schema definition.
//
// Why: the framework redacts attributes marked Sensitive from terraform
// plan output, terraform show, and trace logs. Forgetting to mark a
// password/key/token field as Sensitive leaks its value into every
// terraform invocation's stdout AND into any CI log capture that
// runs `terraform apply`. That's the highest-blast-radius mistake a
// provider author can make for credential handling, second only to
// committing the secret to git.
//
// The invariant is conservative: it only fires on attribute names that
// strongly imply secret values (password, secret, key, token,
// passphrase, etc.). Field names like "name" or "description" never
// trigger. If a sensitive-named field legitimately holds a non-secret
// (e.g., a fingerprint of a key rather than the key itself), add it
// to allowedNonSensitiveFields with rationale.
func TestSensitiveFieldsAreMarkedSensitive(t *testing.T) {
	matches, err := filepath.Glob("../resources/*.go")
	if err != nil {
		t.Fatalf("glob resources: %v", err)
	}

	type finding struct {
		file string
		attr string
	}
	var leaks []finding
	var totalMatched int

	for _, m := range matches {
		base := filepath.Base(m)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		text := string(src)

		for _, attr := range sensitiveAttrBlockRE.FindAllStringSubmatch(text, -1) {
			totalMatched++
			name, body := attr[1], attr[2]
			key := base + "::" + name
			if _, allowed := allowedNonSensitiveFields[key]; allowed {
				continue
			}
			if !strings.Contains(body, "Sensitive:") || !strings.Contains(body, "Sensitive:   true") &&
				!strings.Contains(body, "Sensitive: true") {
				// More lenient check: just look for the substring.
				if !regexp.MustCompile(`Sensitive:\s*true`).MatchString(body) {
					leaks = append(leaks, finding{file: base, attr: name})
				}
			}
		}
	}

	if len(leaks) > 0 {
		var lines []string
		for _, f := range leaks {
			lines = append(lines, "  "+f.file+" :: "+f.attr)
		}
		t.Fatalf("%d sensitive-looking attribute(s) NOT marked Sensitive: true — "+
			"this leaks the value into terraform plan output, terraform show, and "+
			"trace logs on every apply. add `Sensitive: true` to each attribute "+
			"below. if a field LEGITIMATELY holds a non-secret value despite its "+
			"sensitive-sounding name, add it to allowedNonSensitiveFields with "+
			"the honest reason:\n%s",
			len(leaks), strings.Join(lines, "\n"))
	}

	if totalMatched == 0 {
		t.Fatal("invariant scanned no sensitive-named fields — regex broken or no " +
			"sensitive fields exist in the tree. either way that needs investigation.")
	}
	t.Logf("OK: %d sensitive-named schema attributes all marked Sensitive: true", totalMatched)
}
