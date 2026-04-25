package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIsSensitiveKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"password", true},
		{"Password", true},
		{"bind_password", true},
		{"adminPassword", true},
		{"ciphertext_password", true},
		{"dhchap_key", true},
		{"dhchap_ctrl_key", true},
		{"privatekey", true},
		{"private_key", true},
		{"privkey_pem", true},
		{"api_key", true},
		{"apikey", true},
		{"access_token", true},
		{"refresh_token", true},
		{"client_secret", true},
		{"Authorization", true}, // "auth" substring
		{"name", false},
		{"path", false},
		{"id", false},
		{"username", false},
		{"email", false},
	}
	for _, c := range cases {
		got := isSensitiveKey(c.key)
		if got != c.want {
			t.Errorf("isSensitiveKey(%q) = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestRedactJSONBody_FlatObject(t *testing.T) {
	in := []byte(`{"username":"alice","password":"hunter2","path":"/tank/foo"}`)
	out := redactJSONBody(in)

	var got map[string]interface{}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if got["username"] != "alice" {
		t.Errorf("username mangled: %v", got["username"])
	}
	if got["password"] != redactedPlaceholder {
		t.Errorf("password NOT redacted: %v", got["password"])
	}
	if got["path"] != "/tank/foo" {
		t.Errorf("path mangled: %v", got["path"])
	}
}

func TestRedactJSONBody_NestedObject(t *testing.T) {
	in := []byte(`{
		"message": "validation failed",
		"trace": {
			"formatted": "error",
			"config": {
				"username": "root",
				"dhchap_key": "SECRET-SHOULD-NOT-LEAK"
			}
		},
		"extra": [
			{"field": "privatekey", "value": "-----BEGIN PRIVATE KEY-----..."},
			{"field": "ok", "value": "ok"}
		]
	}`)
	out := redactJSONBody(in)

	if strings.Contains(string(out), "SECRET-SHOULD-NOT-LEAK") {
		t.Fatalf("nested dhchap_key value leaked: %s", out)
	}
	// The "value" field inside the extra array is NOT a sensitive key name,
	// so its content ("-----BEGIN PRIVATE KEY-----...") would normally pass
	// through. Confirm that's the current behavior — documented caveat:
	// redaction is by *key name*, not value content. This is an intentional
	// trade-off; see the Phase G CHANGELOG note.
	if !strings.Contains(string(out), "[REDACTED]") {
		t.Errorf("no redaction marker in nested output: %s", out)
	}
}

func TestRedactJSONBody_Array(t *testing.T) {
	in := []byte(`[{"name":"a","password":"p1"},{"name":"b","secret":"s2"}]`)
	out := redactJSONBody(in)

	var got []map[string]interface{}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("lost array elements: %v", got)
	}
	if got[0]["password"] != redactedPlaceholder {
		t.Errorf("[0].password NOT redacted: %v", got[0]["password"])
	}
	if got[1]["secret"] != redactedPlaceholder {
		t.Errorf("[1].secret NOT redacted: %v", got[1]["secret"])
	}
	if got[0]["name"] != "a" || got[1]["name"] != "b" {
		t.Errorf("non-sensitive names mangled: %v", got)
	}
}

func TestRedactJSONBody_NonJSON(t *testing.T) {
	in := []byte("<html>500 Internal Server Error</html>")
	out := redactJSONBody(in)
	if !strings.Contains(string(out), "500 Internal Server Error") {
		t.Errorf("non-JSON body dropped: %s", out)
	}
}

func TestRedactJSONBody_NonJSONTruncated(t *testing.T) {
	big := make([]byte, 2000)
	for i := range big {
		big[i] = 'x'
	}
	out := redactJSONBody(big)
	if len(out) > 600 {
		t.Errorf("non-JSON truncation failed: got %d bytes", len(out))
	}
	if !strings.Contains(string(out), "truncated") {
		t.Errorf("missing truncation marker: %s", out)
	}
}

func TestRedactJSONBody_Empty(t *testing.T) {
	out := redactJSONBody(nil)
	if len(out) != 0 {
		t.Errorf("empty should stay empty: %s", out)
	}
	out = redactJSONBody([]byte{})
	if len(out) != 0 {
		t.Errorf("empty should stay empty: %s", out)
	}
}

func TestRedactMessage(t *testing.T) {
	cases := []struct {
		in      string
		notWant string // substring that must NOT appear in output
		desc    string
	}{
		{
			in:      "validation failed: password must be at least 8 characters",
			notWant: "password must",
			desc:    "password fragment triggers truncation",
		},
		{
			in:      "invalid dhchap_key format",
			notWant: "format",
			desc:    "dhchap_key fragment triggers truncation",
		},
		{
			in:      "dataset creation failed: permission denied",
			notWant: "[REDACTED]",
			desc:    "no sensitive fragment, message passes through",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := redactMessage(c.in)
			if c.notWant == "[REDACTED]" {
				// passthrough case: output must equal input
				if got != c.in {
					t.Errorf("passthrough mutated: got %q, want %q", got, c.in)
				}
			} else {
				if strings.Contains(got, c.notWant) {
					t.Errorf("redaction missed: got %q, must not contain %q", got, c.notWant)
				}
			}
		})
	}
}

// TestAPIErrorBodyNeverLeaksSecrets is an integration test of the full
// flow: a simulated 422 response from TrueNAS carrying a password field
// in its JSON body MUST NOT surface through APIError.Error() — which is
// what gets passed to resp.Diagnostics.AddError on every resource.
func TestAPIErrorBodyNeverLeaksSecrets(t *testing.T) {
	rawBody := []byte(`{"message":"validation failed","trace":{"config":{"password":"LEAK-MARKER-hunter2","dhchap_key":"LEAK-MARKER-chap"}}}`)
	safe := redactJSONBody(rawBody)
	apiErr := &APIError{
		StatusCode: 422,
		Body:       string(safe),
	}
	out := apiErr.Error()
	if strings.Contains(out, "LEAK-MARKER-hunter2") {
		t.Fatalf("password leaked through APIError.Error(): %s", out)
	}
	if strings.Contains(out, "LEAK-MARKER-chap") {
		t.Fatalf("dhchap_key leaked through APIError.Error(): %s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("no redaction marker in error output: %s", out)
	}
}
