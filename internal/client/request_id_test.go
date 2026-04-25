package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
)

// reqIDHexRE matches the 16-char lowercase hex shape produced by
// newRequestID(). Any request ID reaching the wire MUST match.
var reqIDHexRE = regexp.MustCompile(`^[0-9a-f]{16}$`)

// TestNewRequestID_ShapeAndUniqueness verifies the generator emits
// 16-char lowercase hex and that two back-to-back calls do not collide.
// crypto/rand gives us effectively zero collision risk across the
// lifetime of a provider run; this test guards against the generator
// accidentally returning a constant or an empty string (which would
// silently strip the X-Request-ID header via the `if reqID != ""` guard
// in doOnce).
func TestNewRequestID_ShapeAndUniqueness(t *testing.T) {
	a := newRequestID()
	if !reqIDHexRE.MatchString(a) {
		t.Fatalf("newRequestID returned %q, want 16-char lowercase hex", a)
	}
	b := newRequestID()
	if a == b {
		t.Errorf("newRequestID returned the same value twice: %q", a)
	}
}

// TestDoRequest_EmitsXRequestIDHeader makes a real HTTP round-trip to
// an httptest server and asserts that the outgoing request carries a
// valid X-Request-ID header. This is the wire-format test that catches
// regressions where someone removes the header set in doOnce.
func TestDoRequest_EmitsXRequestIDHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-ID")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Get(context.Background(), "/anything"); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got == "" {
		t.Fatal("X-Request-ID header not set on outgoing request")
	}
	if !reqIDHexRE.MatchString(got) {
		t.Errorf("X-Request-ID header shape mismatch: got %q, want 16-char lowercase hex", got)
	}
}

// TestDoRequest_RetriesShareRequestID verifies that when doRequest
// retries an idempotent call after a 503, ALL attempts carry the SAME
// X-Request-ID — the correlation ID belongs to the logical operation,
// not the individual attempt. Operators correlating provider logs with
// server audit entries need to see all attempts under one ID so a
// transient-retry storm doesn't fragment the investigation.
func TestDoRequest_RetriesShareRequestID(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	var count int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, r.Header.Get("X-Request-ID"))
		count++
		if count < 3 {
			// Fail the first two attempts with a retryable status. The
			// client retries idempotent GETs up to RetryPolicy.MaxAttempts.
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"message":"please retry"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Tighten the backoff so the test runs fast.
	c.RetryPolicy = RetryPolicy{MaxAttempts: 5, BaseDelay: 1, MaxDelay: 2}

	if _, err := c.Get(context.Background(), "/anything"); err != nil {
		t.Fatalf("Get after retries: %v", err)
	}

	if len(seen) < 2 {
		t.Fatalf("expected at least 2 attempts, saw %d", len(seen))
	}
	first := seen[0]
	if first == "" {
		t.Fatal("first attempt had no X-Request-ID")
	}
	for i, id := range seen {
		if id != first {
			t.Errorf("attempt %d used a different X-Request-ID: first=%q got=%q — all retries of a logical op must share one ID", i, first, id)
		}
	}
}
