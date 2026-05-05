package wsclient

import (
	"encoding/json"
	"strings"
)

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
// matches isSensitiveKey. Lists and nested maps are walked; scalars
// pass through unchanged when the surrounding key is not sensitive.
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
