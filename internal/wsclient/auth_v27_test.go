package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"
)

// DENIED is new in TrueNAS 26.0 and means the credential is valid but has no
// API access. Before it was handled it fell through to "unexpected
// response_type", which told the operator nothing about the actual fix.
func TestParseLoginExResult_denied(t *testing.T) {
	err := parseLoginExResult([]byte(`{"response_type":"DENIED"}`))
	if err == nil {
		t.Fatal("DENIED was treated as a successful login")
	}
	if strings.Contains(err.Error(), "unexpected response_type") {
		t.Errorf("DENIED fell through to the default branch: %v", err)
	}
	if !strings.Contains(err.Error(), "API access") {
		t.Errorf("diagnostic should name the missing privilege, got: %v", err)
	}
}

func TestParseLoginExResult_knownTypes(t *testing.T) {
	cases := []struct {
		respType string
		wantErr  bool
	}{
		{"SUCCESS", false},
		{"AUTH_ERR", true},
		{"EXPIRED", true},
		{"DENIED", true},
		{"OTP_REQUIRED", true},
		{"REDIRECT", true},
	}
	for _, tc := range cases {
		t.Run(tc.respType, func(t *testing.T) {
			err := parseLoginExResult([]byte(`{"response_type":"` + tc.respType + `"}`))
			if tc.wantErr && err == nil {
				t.Error("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			// Only a genuinely unknown type may use the default branch.
			if err != nil && strings.Contains(err.Error(), "unexpected response_type") {
				t.Errorf("%s is a documented type but hit the default branch", tc.respType)
			}
		})
	}
}

func TestParseLoginExResult_stillRejectsUnknown(t *testing.T) {
	err := parseLoginExResult([]byte(`{"response_type":"SOMETHING_NEW"}`))
	if err == nil || !strings.Contains(err.Error(), "unexpected response_type") {
		t.Errorf("an unknown type must still be rejected loudly, got: %v", err)
	}
}

// --- the v27 legacy-handshake guard ---

// legacyAuthServer models a TrueNAS that either has the legacy handshake or,
// like 27.0, answers method-not-found for it.
func legacyAuthServer(t *testing.T, hasLegacy bool, sawLegacy *bool) *TestServer {
	t.Helper()
	return NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "auth.login_with_api_key":
			if sawLegacy != nil {
				*sawLegacy = true
			}
			if !hasLegacy {
				// What 27.0 returns: the method is @private, so it is
				// registered on no endpoint.
				return nil, &RPCError{Code: CodeMethodNotFound, Message: "auth.login_with_api_key"}
			}
			return true, nil
		case "auth.login_ex":
			return map[string]interface{}{"response_type": "SUCCESS"}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	}, TestServerOptions{SkipAuth: true})
}

// On 27 the method is gone, so the bare -32601 must be translated into
// something that names the actual fix.
func TestAuthenticate_explainsLegacyRemovalOnV27(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := legacyAuthServer(t, false, nil)

	_, err := NewWithUsername(ctx, ts.URL(), "test-api-key", "", true)
	if err == nil {
		t.Fatal("a missing legacy handshake was treated as a successful login")
	}
	for _, want := range []string{"27.0", "username"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("diagnostic should mention %q, got: %v", want, err)
		}
	}
}

// Only method-not-found gets the v27 explanation. A wrong key must keep its
// own error, or every auth failure would blame the TrueNAS version.
func TestAuthenticate_otherAuthErrorsAreNotBlamedOnV27(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EACCES] bad key"}
	}, TestServerOptions{SkipAuth: true})
	_, err := NewWithUsername(ctx, ts.URL(), "test-api-key", "", true)
	if err == nil {
		t.Fatal("a rejected key was treated as success")
	}
	if strings.Contains(err.Error(), "27.0") {
		t.Errorf("a bad key was blamed on the TrueNAS version: %v", err)
	}
}

// Servers that still have the method must keep working untouched.
func TestAuthenticate_legacyHandshakeStillWorksPreV27(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawLegacy bool
	ts := legacyAuthServer(t, true, &sawLegacy)
	if _, err := NewWithUsername(ctx, ts.URL(), "test-api-key", "", true); err != nil {
		t.Fatalf("legacy handshake refused on a server that has it: %v", err)
	}
	if !sawLegacy {
		t.Error("the legacy method was not used")
	}
}

// A configured username uses login_ex, which survives 27, so the guard must
// not fire.
func TestAuthenticate_usernameUsesLoginExOnV27(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawLegacy bool
	ts := legacyAuthServer(t, false, &sawLegacy)
	if _, err := NewWithUsername(ctx, ts.URL(), "test-api-key", "someuser", true); err != nil {
		t.Fatalf("login_ex refused on 27.0: %v", err)
	}
	if sawLegacy {
		t.Error("the legacy method was used despite a configured username")
	}
}

// With a username configured the login_ex error must surface untouched; the
// v27 explanation applies only to the legacy path.
func TestAuthenticate_loginExErrorSurfacesUnwrapped(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EACCES] nope"}
	}, TestServerOptions{SkipAuth: true})
	_, err := NewWithUsername(ctx, ts.URL(), "test-api-key", "someuser", true)
	if err == nil {
		t.Fatal("a failed login_ex was treated as success")
	}
	if strings.Contains(err.Error(), "27.0") {
		t.Errorf("a login_ex failure was blamed on the v27 removal: %v", err)
	}
}
