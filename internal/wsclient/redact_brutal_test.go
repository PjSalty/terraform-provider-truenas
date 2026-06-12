package wsclient

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRedact_ExhaustivelyEnumerateSensitiveKeys asserts that EVERY key
// in the schema-declared `Sensitive: true` registry triggers redaction.
// The bug pattern this catches: a new resource ships a sensitive
// attribute with a name that doesn't include any of the
// sensitiveKeyFragments substrings, e.g. a future TrueNAS API adds
// "vault_unlock_phrase" or "kmip_handshake_pem". Schema marks it
// sensitive (which keeps it out of the Terraform plan output) but
// the client's redactor doesn't, so it leaks into logs / error
// strings / the API request trace.
//
// The list below mirrors `sensitiveKeyRegistry` in the schema-
// invariant test under internal/provider/, kept synced manually for
// now. CI flag: if the schema list grows, add the new attribute name
// here too.
func TestRedact_ExhaustivelyEnumerateSensitiveKeys(t *testing.T) {
	// Every sensitive attribute name across the resource set,
	// canonical form (snake_case as TrueNAS JSON would emit).
	knownSensitiveAttrs := []string{
		"password",
		"passwd",
		"smb_password",
		"ldap_bind_password",
		"idp_bind_password",
		"smtp_password",
		"secret",
		"peersecret",
		"client_secret",
		"webhook_secret",
		"shared_secret",
		"slack_secret",
		"jwt_secret",
		"hmac_secret",
		"privatekey",
		"private_key",
		"privkey",
		"ssh_private_key",
		"sftp_private_key",
		"acme_account_key",
		"dhchap_key",
		"dhchap_ctrl_key",
		"apikey",
		"api_key",
		"telegram_api_key",
		"pushover_api_key",
		"healthchecks_api_key",
		"token",
		"oauth_token",
		"refresh_token",
		"access_token",
		"session_token",
		"bearer",
		"auth_token",
		"vault_token",
		"authorization",
		"credentials",
		"google_credentials",
		"aws_credentials",
		"passphrase",
		"key_material",
		"shared_key",
		"preshared_key",
		"pre_shared_key",
	}

	for _, k := range knownSensitiveAttrs {

		t.Run(k, func(t *testing.T) {
			if !isSensitiveKey(k) {
				t.Errorf("isSensitiveKey(%q) = false; the schema marks this attribute "+
					"Sensitive: true but the redactor would miss it. add a matching "+
					"fragment to sensitiveKeyFragments in internal/client/redact.go.", k)
			}
		})
	}
}

// TestRedact_NestedAndArrayShapes asserts the JSON walker reaches
// secrets buried under arbitrary depth and array indexes. SCALE 25.10+
// nests sensitive fields under `attributes` and array entries for
// alert services, cloud-sync credentials, and OIDC client configs.
// A walker that bails after the first level would leak these.
func TestRedact_NestedAndArrayShapes(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		shouldNot string // substring that must NOT remain in output
	}{
		{
			name:      "top-level password",
			body:      `{"password":"s3cret"}`,
			shouldNot: "s3cret",
		},
		{
			name:      "nested object password",
			body:      `{"attributes":{"password":"s3cret"}}`,
			shouldNot: "s3cret",
		},
		{
			name:      "deeply nested password",
			body:      `{"a":{"b":{"c":{"d":{"password":"s3cret"}}}}}`,
			shouldNot: "s3cret",
		},
		{
			name:      "array of objects with secret",
			body:      `[{"name":"x","secret":"s3cret"}]`,
			shouldNot: "s3cret",
		},
		{
			name:      "mixed array under nested key",
			body:      `{"credentials":[{"api_key":"s3cret"},{"token":"s3cret"}]}`,
			shouldNot: "s3cret",
		},
		{
			name:      "uppercase key",
			body:      `{"PASSWORD":"s3cret"}`,
			shouldNot: "s3cret",
		},
		{
			name:      "mixed-case key",
			body:      `{"ApiKey":"s3cret"}`,
			shouldNot: "s3cret",
		},
		{
			name:      "key containing substring",
			body:      `{"my_oidc_client_secret":"s3cret"}`,
			shouldNot: "s3cret",
		},
		{
			name:      "value is itself a JSON string",
			body:      `{"settings_json":"{\"password\":\"s3cret\"}"}`,
			shouldNot: "s3cret", // attributes.password inside a JSON-string value
		},
	}
	for _, c := range cases {

		t.Run(c.name, func(t *testing.T) {
			out := redactJSONBody([]byte(c.body))
			if strings.Contains(string(out), c.shouldNot) {
				t.Errorf("secret leaked through redactor:\n  in:  %s\n  out: %s", c.body, out)
			}
		})
	}
}

// TestRedact_RoundTripValidJSON asserts the redactor produces valid
// JSON output even on weird inputs. A malformed output would crash
// downstream loggers that re-parse, or worse, leak the raw input.
func TestRedact_RoundTripValidJSON(t *testing.T) {
	inputs := [][]byte{
		[]byte(`{"password":"a"}`),
		[]byte(`{}`),
		[]byte(`[]`),
		[]byte(`null`),
		[]byte(`[1,2,3]`),
		[]byte(`{"a":[1,{"password":"x"},3]}`),
		[]byte(`{"a":{"b":{"c":{"password":"deep"}}}}`),
		[]byte(`{"password":null}`),
		[]byte(`{"password":42}`),
		[]byte(`{"password":[1,2,3]}`),
		[]byte(`{"password":{"nested":"value"}}`),
	}
	for _, in := range inputs {
		out := redactJSONBody(in)
		// Output must be valid JSON
		var v interface{}
		if err := json.Unmarshal(out, &v); err != nil {
			t.Errorf("redactor produced invalid JSON for %q: %s\n  err: %v", in, out, err)
		}
	}
}

// TestRedact_MessageStrings asserts redactMessage drops common
// secret-bearing substrings from error message payloads, the place
// where API errors get re-surfaced to the user and to logs.
func TestRedact_MessageStrings(t *testing.T) {
	cases := []struct {
		name string
		in   string
		hide string
	}{
		{"bearer token", "Authorization: Bearer abc123def456", "abc123def456"},
		{"api-key header", "X-API-Key: deadbeef", "deadbeef"},
		{"password in url", "https://user:p4ss@host/path", "p4ss"},
		{"json fragment", `error: invalid password "myp4ssword"`, "myp4ssword"},
	}
	for _, c := range cases {

		t.Run(c.name, func(t *testing.T) {
			out := redactMessage(c.in)
			if strings.Contains(out, c.hide) {
				t.Errorf("secret leaked through message redactor:\n  in:  %s\n  out: %s",
					c.in, out)
			}
		})
	}
}
