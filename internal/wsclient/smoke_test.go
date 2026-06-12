package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestSmoke_RoundTrip is the minimum viable end-to-end test: dial,
// authenticate, call a single method, get a typed response back. If
// this test fails the whole package is broken; if it passes the
// architecture is wired up correctly.
func TestSmoke_RoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "system.info" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: "unexpected: " + method}
		}
		return map[string]interface{}{
			"hostname": "test-truenas",
			"version":  "TrueNAS-SCALE-25.04.2",
		}, nil
	})

	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := c.Call(ctx, "system.info", nil, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}

	var info struct {
		Hostname string `json:"hostname"`
		Version  string `json:"version"`
	}
	if err := json.Unmarshal(result, &info); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if info.Hostname != "test-truenas" {
		t.Errorf("hostname: got %q, want %q", info.Hostname, "test-truenas")
	}
	if info.Version != "TrueNAS-SCALE-25.04.2" {
		t.Errorf("version: got %q, want %q", info.Version, "TrueNAS-SCALE-25.04.2")
	}
}

// TestSmoke_RPCError verifies that a server-side RPCError surfaces as
// a typed *RPCError on the client side, scrubbed and unwrapped.
func TestSmoke_RPCError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{
			Code:    CodeMethodCallError,
			Message: "Validation failed",
			Data:    json.RawMessage(`{"errname":"ValidationErrors","reason":"object 'tank/missing' does not exist"}`),
		}
	})

	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Call(ctx, "pool.dataset.get_instance", []interface{}{"tank/missing"}, CallOptions{Idempotent: true})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error is not *RPCError: %T %v", err, err)
	}
	if rpcErr.Code != CodeMethodCallError {
		t.Errorf("code: got %d, want %d", rpcErr.Code, CodeMethodCallError)
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound returned false for ValidationErrors/does not exist; want true")
	}
}

// TestSmoke_ReadOnly verifies the read-only safety rail blocks a
// mutating method before it hits the server.
func TestSmoke_ReadOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	called := false
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		called = true
		return nil, nil
	})

	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.ReadOnly = true

	_, err = c.Call(ctx, "pool.dataset.create", []interface{}{}, CallOptions{})
	if !errors.Is(err, ErrReadOnly) {
		t.Fatalf("expected ErrReadOnly, got %v", err)
	}
	if called {
		t.Error("server handler was called despite ReadOnly=true")
	}
}

// TestSmoke_DestroyProtection verifies the destroy-protection rail
// blocks a destructive method.
func TestSmoke_DestroyProtection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return true, nil
	})

	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.DestroyProtection = true

	// pool.dataset.delete is destructive — must be blocked.
	_, err = c.Call(ctx, "pool.dataset.delete", []interface{}{"tank/foo"}, CallOptions{})
	if !errors.Is(err, ErrDestroyProtected) {
		t.Fatalf("expected ErrDestroyProtected, got %v", err)
	}

	// pool.dataset.update is mutating but NOT destructive — must pass.
	_, err = c.Call(ctx, "pool.dataset.update", []interface{}{"tank/foo", map[string]any{}}, CallOptions{})
	if err != nil {
		t.Fatalf("expected pool.dataset.update to pass with DestroyProtection=true, got %v", err)
	}
}

// TestSmoke_AuthFailure verifies a server returning false from
// auth.login_with_api_key surfaces as an authentication error.
func TestSmoke_AuthFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "auth.login_with_api_key" {
			return false, nil
		}
		return nil, nil
	}, TestServerOptions{SkipAuth: true})

	_, err := New(ctx, ts.URL(), "wrong-key", true)
	if err == nil {
		t.Fatal("expected auth failure, got nil")
	}
	if !strings.Contains(err.Error(), "auth.login_with_api_key") {
		t.Errorf("error does not mention auth.login_with_api_key: %v", err)
	}
}
