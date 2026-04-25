package client

import (
	"encoding/json"
	"strings"
)

// sensitiveKeyFragments is the set of JSON field-name fragments that, when
// they appear as a key in any object (at any depth), cause that field's
// value to be replaced with "[REDACTED]" before the body is stored on an
// APIError or surfaced to diagnostics.
//
// The match is case-insensitive and substring-based, so "password",
// "bind_password", "adminPassword" and "ciphertext_password" all match
// the "password" fragment. The intent is to be aggressive: a false
// positive just hides a non-secret, while a false negative leaks a
// real secret into terraform output and state files.
//
// Derived from the union of every schema attribute marked Sensitive: true
// in internal/resources/*.go plus the canonical HTTP auth-related names.
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

// isSensitiveKey reports whether a JSON key looks sensitive. Case-insensitive,
// substring match. Guarantees that key "Authorization" (contains "auth")
// and "idp_bind_password" (contains "password") both trigger redaction.
func isSensitiveKey(key string) bool {
	low := strings.ToLower(key)
	for _, frag := range sensitiveKeyFragments {
		if strings.Contains(low, frag) {
			return true
		}
	}
	return false
}

// redactJSONBody takes a raw response body (which may be JSON or not),
// recursively walks any JSON it can parse, replaces sensitive field
// values with "[REDACTED]", and returns the result re-serialized. On
// non-JSON bodies it truncates to 512 bytes and returns a placeholder
// marker so a raw binary or HTML error page can't dump unbounded data
// into a diagnostic.
//
// This is called in doOnce on every non-2xx response BEFORE constructing
// the APIError, so that APIError.Body (which is what surfaces through
// err.Error() and ultimately into resp.Diagnostics.AddError) is
// guaranteed secret-free.
func redactJSONBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		if len(body) > 512 {
			return append([]byte("[non-JSON error body, truncated] "), body[:512]...)
		}
		return body
	}
	redacted := walkRedact(parsed)
	// walkRedact only produces Go values that came from json.Unmarshal
	// (maps[string]interface{}, []interface{}, strings, numbers, bools,
	// nil), so re-marshaling is guaranteed to succeed. The error return
	// of json.Marshal exists only for types it can't serialize (chans,
	// funcs, unsupported ints) which walkRedact never produces.
	out, _ := json.Marshal(redacted)
	return out
}

// walkRedact recursively replaces the value of any map entry whose key
// matches isSensitiveKey. Lists and nested maps are walked; scalars are
// passed through unchanged when the surrounding key is not sensitive.
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

// redactMessage scrubs a parsed error "message" string for secret-looking
// substrings of the form `<sensitive-key>=<value>` or `'<sensitive-key>':
// '<value>'`. TrueNAS middlewared error messages occasionally echo back
// offending request fields in their Pydantic validation output; this
// catches the common case. Best-effort — not a substitute for key-based
// body redaction, which is authoritative.
func redactMessage(msg string) string {
	if msg == "" {
		return msg
	}
	low := strings.ToLower(msg)
	for _, frag := range sensitiveKeyFragments {
		if !strings.Contains(low, frag) {
			continue
		}
		// If the message mentions a sensitive key name at all, redact
		// aggressively by returning the first 120 chars of the message
		// up to the fragment, then a placeholder. This trades diagnostic
		// quality for safety; operators can still see the HTTP status
		// and the leading part of the error.
		idx := strings.Index(low, frag)
		if idx > 0 {
			return msg[:idx] + redactedPlaceholder
		}
		return redactedPlaceholder
	}
	return msg
}
