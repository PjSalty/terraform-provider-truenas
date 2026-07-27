package wsclient

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// loginExRecorder captures the auth calls the client makes so tests
// can assert mechanism selection and exact wire payloads.
type loginExRecorder struct {
	mu      sync.Mutex
	methods []string
	payload map[string]interface{}
}

func newLoginExServer(t *testing.T, respond func() (interface{}, *RPCError)) (*TestServer, *loginExRecorder) {
	t.Helper()
	rec := &loginExRecorder{}
	ts := NewTestServerWithOptions(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		rec.mu.Lock()
		rec.methods = append(rec.methods, method)
		if method == "auth.login_ex" && len(params) == 1 {
			if m, ok := params[0].(map[string]interface{}); ok {
				rec.payload = m
			}
		}
		rec.mu.Unlock()
		switch method {
		case "auth.login_ex", "auth.login_with_api_key":
			return respond()
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	}, TestServerOptions{SkipAuth: true})
	return ts, rec
}

// a configured username must select auth.login_ex with the exact
// API_KEY_PLAIN payload and never touch the legacy call.
func TestAuthLoginEx_MechanismAndPayload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ts, rec := newLoginExServer(t, func() (interface{}, *RPCError) {
		return map[string]interface{}{"response_type": "SUCCESS"}, nil
	})
	c, err := NewWithUsername(ctx, ts.URL(), "k3y", "root", true)
	if err != nil {
		t.Fatalf("NewWithUsername: %v", err)
	}
	defer func() { _ = c.Close() }()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, m := range rec.methods {
		if m == "auth.login_with_api_key" {
			t.Error("legacy auth call used despite a configured username")
		}
	}
	if rec.payload["mechanism"] != "API_KEY_PLAIN" || rec.payload["username"] != "root" || rec.payload["api_key"] != "k3y" {
		t.Fatalf("login_ex payload wrong: %v", rec.payload)
	}
}

// an empty username must keep the legacy handshake byte-for-byte.
func TestAuthLoginEx_EmptyUsernameUsesLegacy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ts, rec := newLoginExServer(t, func() (interface{}, *RPCError) {
		return true, nil
	})
	c, err := NewWithUsername(ctx, ts.URL(), "k3y", "", true)
	if err != nil {
		t.Fatalf("NewWithUsername(empty): %v", err)
	}
	defer func() { _ = c.Close() }()
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, m := range rec.methods {
		if m == "auth.login_ex" {
			t.Error("login_ex used without a username")
		}
	}
}

// every non-SUCCESS response_type maps to an actionable error.
func TestAuthLoginEx_ResponseTypes(t *testing.T) {
	cases := []struct {
		name     string
		resp     interface{}
		wantPart string
	}{
		{"auth err", map[string]interface{}{"response_type": "AUTH_ERR"}, "authentication failed"},
		{"expired", map[string]interface{}{"response_type": "EXPIRED"}, "expired"},
		{"otp", map[string]interface{}{"response_type": "OTP_REQUIRED", "username": "root"}, "one-time password"},
		{"redirect", map[string]interface{}{"response_type": "REDIRECT", "urls": []string{"https://other/"}}, "redirected"},
		{"unknown type", map[string]interface{}{"response_type": "WAT"}, "unexpected response_type"},
		{"garbage shape", "nonsense", "unexpected result shape"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ts, _ := newLoginExServer(t, func() (interface{}, *RPCError) {
				return tc.resp, nil
			})
			_, err := NewWithUsername(ctx, ts.URL(), "k3y", "root", true)
			if err == nil {
				t.Fatal("want auth error, got success")
			}
			if !strings.Contains(err.Error(), tc.wantPart) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantPart)
			}
		})
	}
}

// the rate-limit retry loop must cover the login_ex path exactly as it
// covers the legacy one.
func TestAuthLoginEx_RateLimitRetries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var mu sync.Mutex
	calls := 0
	ts, _ := newLoginExServer(t, func() (interface{}, *RPCError) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			return nil, &RPCError{Code: CodeMethodCallError, Message: "[EBUSY] Rate Limit Exceeded"}
		}
		return map[string]interface{}{"response_type": "SUCCESS"}, nil
	})
	c, err := NewWithUsername(ctx, ts.URL(), "k3y", "root", true)
	if err != nil {
		t.Fatalf("expected retry to succeed: %v", err)
	}
	defer func() { _ = c.Close() }()
	mu.Lock()
	defer mu.Unlock()
	if calls < 2 {
		t.Fatalf("rate-limited call was not retried (calls=%d)", calls)
	}
}

// the default test server must auto-answer login_ex the way it
// auto-answers the legacy call, so resource tests can dial with a
// username without special-casing auth.
func TestAuthLoginEx_DefaultTestServerShortCircuit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ts := NewTestServer(t, nil)
	c, err := NewWithUsername(ctx, ts.URL(), "k3y", "root", true)
	if err != nil {
		t.Fatalf("NewWithUsername vs default server: %v", err)
	}
	_ = c.Close()
}
