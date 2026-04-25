package client_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestReadOnly_BlocksPOST verifies that a POST attempt on a client in
// read-only mode returns ErrReadOnly without making any network call.
func TestReadOnly_BlocksPOST(t *testing.T) {
	var calls int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	})
	_, c := newTestServer(t, handler)
	c.ReadOnly = true

	_, err := c.Post(context.Background(), "/some/path", map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, client.ErrReadOnly) {
		t.Errorf("errors.Is(err, ErrReadOnly) = false; err = %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("server received %d calls, want 0 (read-only must not hit the network)", got)
	}
}

// TestReadOnly_BlocksPUT verifies that PUT is blocked even though it is
// idempotent. Idempotency is a retry property, not a safety property.
func TestReadOnly_BlocksPUT(t *testing.T) {
	var calls int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	})
	_, c := newTestServer(t, handler)
	c.ReadOnly = true

	_, err := c.Put(context.Background(), "/some/path", map[string]string{"k": "v"})
	if err == nil || !errors.Is(err, client.ErrReadOnly) {
		t.Fatalf("want ErrReadOnly, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("server received %d calls, want 0", got)
	}
}

// TestReadOnly_BlocksDELETE verifies that DELETE is blocked.
func TestReadOnly_BlocksDELETE(t *testing.T) {
	var calls int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	})
	_, c := newTestServer(t, handler)
	c.ReadOnly = true

	_, err := c.Delete(context.Background(), "/some/path")
	if err == nil || !errors.Is(err, client.ErrReadOnly) {
		t.Fatalf("want ErrReadOnly, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("server received %d calls, want 0", got)
	}
}

// TestReadOnly_AllowsGET verifies that GET passes through untouched.
func TestReadOnly_AllowsGET(t *testing.T) {
	var calls int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`"ok"`))
	})
	_, c := newTestServer(t, handler)
	c.ReadOnly = true

	body, err := c.Get(context.Background(), "/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server received %d calls, want 1", got)
	}
	if string(body) != `"ok"` {
		t.Errorf("body = %q, want \"ok\"", string(body))
	}
}

// TestReadOnly_Disabled verifies that when ReadOnly is false (the default),
// mutating methods pass through normally.
func TestReadOnly_Disabled(t *testing.T) {
	var calls int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	})
	_, c := newTestServer(t, handler)
	// ReadOnly intentionally left false (zero value)

	if _, err := c.Post(context.Background(), "/p", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server received %d calls, want 1", got)
	}
}

// TestReadOnly_ErrorFormat verifies the error message includes the attempted
// method + path so a operator reading the Terraform error output can tell
// exactly which resource tried to mutate.
func TestReadOnly_ErrorFormat(t *testing.T) {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	c.ReadOnly = true

	_, err := c.Delete(context.Background(), "/dataset/tank%2Fimportant")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "DELETE") {
		t.Errorf("error missing method: %v", err)
	}
	if !strings.Contains(msg, "/dataset/tank%2Fimportant") {
		t.Errorf("error missing path: %v", err)
	}
	if !strings.Contains(msg, "read-only") {
		t.Errorf("error missing read-only marker: %v", err)
	}
}

// TestReadOnly_NilClientSafe verifies that calling checkReadOnly through
// the exported surface is safe even for the zero value — specifically,
// constructing a Client with ReadOnly=false and no calls must not panic.
func TestReadOnly_NilClientSafe(t *testing.T) {
	// Build a minimal client and verify zero-value ReadOnly works.
	c, err := client.New("http://example.invalid", "k")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.ReadOnly {
		t.Error("new client defaulted to ReadOnly=true; want false")
	}
}
