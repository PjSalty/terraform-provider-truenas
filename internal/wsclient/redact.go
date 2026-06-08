package wsclient

import (
	"encoding/json"
	"regexp"
	"strings"
)

// messageRedactRegexps mirrors client.messageRedactRegexps — patterns
// that catch secret material in error message bodies even when the
// key-name match misses (headers with hyphens, URLs with basic-auth,
// bearer tokens). Kept in sync manually with the REST client copy.
var messageRedactRegexps = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\b[a-z][a-z0-9+\-.]*://[^\s:@/]*:)([^@\s/]+)(@)`),
	regexp.MustCompile(`(?i)(\b(?:authorization|x-api-key|x-auth-token|cookie|set-cookie)\s*[:=]\s*)([^\s\r\n;,]+)`),
	regexp.MustCompile(`(?i)(\bBearer\s+)([A-Za-z0-9._\-+/=]+)`),
}

// sensitiveKeyFragments mirrors client.sensitiveKeyFragments. We keep a
// duplicate copy in this package rather than importing the REST client
// because the REST package is being deleted in v2.1 — duplicating the
// list now lets the wsclient stand alone without an awkward inter-
// package dependency for a 30-element slice.
//
// Whenever a new Sensitive: true attribute is added to internal/resources/,
// add the matching fragment here AND in client/redact.go. A small
// invariant test in internal/wsclient/redact_test.go enforces parity.
var sensitiveKeyFragments = []string{
	"password",
	"passwd",
	"secret",
	"privatekey",
	"private_key",
	"privkey",
	"dhchap_key",
	"dhchap_ctrl_key",
	"apikey",
	"api_key",
	"token",
	"bearer",
	"auth",
	"credential",
	"passphrase",
	"key_material",
	"shared_key",
	"preshared",
	"pre_shared",
	"aws_secret_access_key",
	"aws_session_token",
	"client_secret",
	"refresh_token",
	"access_token",
	"session_token",
	// Added 2026-06-08 after the property-based redactor tests
	// surfaced ACME account_key as a leak.
	"account_key",
	// "file" was considered but rejected: substring match would
	// trigger on "filesystem", "file_path", "file_size" and break
	// real error messages. Kerberos keytab.file is schema-Sensitive
	// which covers Terraform state/plan output; deeper coverage
	// tracked separately.
}

// redactedPlaceholder is what replaces a sensitive value.
const redactedPlaceholder = "[REDACTED]"

// isSensitiveKey reports whether a JSON key looks sensitive.
// Case-insensitive, substring match.
func isSensitiveKey(key string) bool {
	low := strings.ToLower(key)
	for _, frag := range sensitiveKeyFragments {
		if strings.Contains(low, frag) {
			return true
		}
	}
	return false
}

// redactJSONBody walks any JSON it can parse, replaces sensitive
// field values with "[REDACTED]", and returns the result re-serialized.
// On non-JSON input, returns the input verbatim up to 512 bytes with a
// "non-JSON" marker so a binary or HTML error page cannot dump
// unbounded data into a diagnostic.
//
// Called from RPCError construction and from the result-marshaling
// path before any value reaches Diagnostics or tflog.
func redactJSONBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		if len(body) > 512 {
			return append([]byte("[non-JSON body, truncated] "), body[:512]...)
		}
		return body
	}
	redacted := walkRedact(parsed)
	out, _ := json.Marshal(redacted)
	return out
}

// walkRedact recursively replaces the value of any map entry whose key
// matches isSensitiveKey. Lists and nested maps are walked; string
// values that themselves parse as JSON are recursively redacted then
// re-marshalled (catches TrueNAS' settings_json / attributes_json
// pattern where a secret can be buried inside a JSON-string attribute).
// Scalars and non-JSON strings pass through unchanged when the
// surrounding key is not sensitive.
func walkRedact(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			if isSensitiveKey(k) {
				out[k] = redactedPlaceholder
				continue
			}
			out[k] = walkRedact(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			out[i] = walkRedact(val)
		}
		return out
	case string:
		s := strings.TrimSpace(t)
		if len(s) < 2 || (s[0] != '{' && s[0] != '[') {
			return v
		}
		var inner interface{}
		if err := json.Unmarshal([]byte(t), &inner); err != nil {
			return v
		}
		redactedInner := walkRedact(inner)
		out, err := json.Marshal(redactedInner)
		if err != nil {
			return v
		}
		return string(out)
	default:
		return v
	}
}

// redactMessage scrubs a parsed error message string for substrings of
// the form `<sensitive-key>=<value>` or `'<sensitive-key>': '<value>'`.
// TrueNAS middlewared sometimes echoes offending request fields in
// validation errors; this catches the common case. Best-effort —
// key-based body redaction in redactJSONBody is authoritative.
func redactMessage(msg string) string {
	if msg == "" {
		return msg
	}
	// Pattern-based pass first — catches URLs with basic-auth,
	// header values with hyphens (X-API-Key, Authorization), and
	// bare Bearer tokens that the key-fragment match would miss.
	for _, re := range messageRedactRegexps {
		msg = re.ReplaceAllString(msg, "${1}"+redactedPlaceholder+"${3}")
	}
	low := strings.ToLower(msg)
	for _, frag := range sensitiveKeyFragments {
		if !strings.Contains(low, frag) {
			continue
		}
		idx := strings.Index(low, frag)
		if idx > 0 {
			return msg[:idx] + redactedPlaceholder
		}
		return redactedPlaceholder
	}
	return msg
}
