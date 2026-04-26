package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// failReader is an io.Reader that always returns the given error. Used
// to exercise newRequestIDFrom's crypto/rand fallback branch — which
// cannot be hit by passing rand.Reader because that reader never fails
// on a healthy system.
type failReader struct{ err error }

func (f failReader) Read(p []byte) (int, error) { return 0, f.err }

// TestNewRequestIDFrom_Fallback exercises the entropy-source-failure
// branch of newRequestIDFrom. When the provided io.Reader fails, the
// function must return a timestamp-derived ID rather than an empty
// string, so the X-Request-ID header is never silently dropped.
func TestNewRequestIDFrom_Fallback(t *testing.T) {
	id := newRequestIDFrom(failReader{err: io.ErrUnexpectedEOF})
	if id == "" {
		t.Fatal("newRequestIDFrom returned empty string on entropy failure")
	}
	// Timestamp-hex IDs are NOT exactly 16 chars (the crypto/rand form is).
	// A strict != 16 check would be brittle in the 2038-era, but we can
	// assert the string is non-empty and hex-compatible.
	for _, r := range id {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			t.Errorf("fallback ID contains non-hex char %q: %q", r, id)
		}
	}
}

// TestNewRequestIDFrom_Success exercises the happy path on a reader
// that successfully produces 8 bytes of entropy. The returned ID must
// be 16 hex chars.
func TestNewRequestIDFrom_Success(t *testing.T) {
	// Deterministic 8-byte source.
	src := strings.NewReader("12345678")
	id := newRequestIDFrom(src)
	if len(id) != 16 {
		t.Errorf("success path should return 16 hex chars, got %d: %q", len(id), id)
	}
	if id != "3132333435363738" {
		t.Errorf("expected hex of '12345678', got %q", id)
	}
}

// TestDefaultRetryPolicy asserts the defaults match the documented
// contract so operators who don't override any knob get sensible
// behavior out of the box.
func TestDefaultRetryPolicy(t *testing.T) {
	p := DefaultRetryPolicy()
	if p.MaxAttempts != 5 {
		t.Errorf("MaxAttempts: got %d, want 5", p.MaxAttempts)
	}
	if p.BaseDelay.Milliseconds() != 500 {
		t.Errorf("BaseDelay: got %v, want 500ms", p.BaseDelay)
	}
	if p.MaxDelay.Seconds() != 10 {
		t.Errorf("MaxDelay: got %v, want 10s", p.MaxDelay)
	}
}

// TestAPIErrorError exercises both branches of APIError.Error():
//   - Message set → "TrueNAS API error (HTTP N): <message>"
//   - Message empty → "TrueNAS API error (HTTP N): <body>"
func TestAPIErrorError(t *testing.T) {
	withMsg := &APIError{StatusCode: 422, Message: "validation failed", Body: "ignored-when-msg-set"}
	if got := withMsg.Error(); !strings.Contains(got, "validation failed") || strings.Contains(got, "ignored") {
		t.Errorf("message branch wrong: %q", got)
	}

	noMsg := &APIError{StatusCode: 500, Body: "internal server error"}
	if got := noMsg.Error(); !strings.Contains(got, "internal server error") {
		t.Errorf("body fallback branch wrong: %q", got)
	}
}

// TestClientDelete_NoBody exercises Client.Delete (which is a thin
// wrapper around doRequest(DELETE, path, nil)) against an httptest
// server. Closes the 0% coverage on Delete that the CI gate flagged.
func TestClientDelete_NoBody(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, err := New(srv.URL, "dummy")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.Delete(context.Background(), "/some/path"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

// TestClientDeleteWithBody exercises Client.DeleteWithBody — the
// variant used by DELETE endpoints that carry a JSON body (e.g.
// /app/id/{id} on TrueNAS SCALE, which accepts a
// {"remove_images": bool} payload). Closes the 0% coverage on
// DeleteWithBody.
func TestClientDeleteWithBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "dummy")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	payload := map[string]bool{"remove_images": true}
	if _, err := c.DeleteWithBody(context.Background(), "/app/id/x", payload); err != nil {
		t.Fatalf("DeleteWithBody: %v", err)
	}
	if !strings.Contains(gotBody, "remove_images") {
		t.Errorf("expected body to include remove_images, got %q", gotBody)
	}
}

// TestRedactJSONBody_MessageAtStart covers the `idx > 0` vs idx == 0
// branch of redactMessage: when the sensitive fragment appears at the
// very start of the message, the function returns redactedPlaceholder
// alone (the "idx > 0" branch doesn't fire). Closes the remaining
// <100% gap on redactMessage.
func TestRedactJSONBody_MessageAtStart(t *testing.T) {
	msg := "password validation failed"
	out := redactMessage(msg)
	if out != "[REDACTED]" {
		t.Errorf("fragment-at-start should return bare marker, got %q", out)
	}
}

// TestRedactMessage_EmptyString covers the early-return for empty msg
// which the existing TestRedactMessage cases don't hit.
func TestRedactMessage_EmptyString(t *testing.T) {
	if got := redactMessage(""); got != "" {
		t.Errorf("empty msg should pass through empty, got %q", got)
	}
}

// TestWaitIfJobResponse_NonJobResponse covers the "sync-completed"
// branch of waitIfJobResponse where the server returned something
// that definitively isn't a job ID integer (an object, a string, an
// array). Returning nil is the documented dual-response contract.
// Note: JSON `null` silently unmarshals into int as 0, so we don't
// use it as a non-int sentinel — the production semantics here are
// "if you can't parse as int, treat as sync-complete", and `null`
// DOES parse as int(0), which means it would incorrectly try to
// wait for job 0. The production code would also misbehave on a
// literal `null` response; in practice TrueNAS middlewared never
// returns a JSON null body for sync-completed ops.
func TestWaitIfJobResponse_NonJobResponse(t *testing.T) {
	c, err := New("http://127.0.0.1:1", "dummy")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Object body — definitively not an int.
	if err := c.waitIfJobResponse(context.Background(), []byte(`{"done": true}`), "test-op"); err != nil {
		t.Errorf("object response should return nil, got: %v", err)
	}
	// String body — definitively not an int.
	if err := c.waitIfJobResponse(context.Background(), []byte(`"sync-complete"`), "test-op"); err != nil {
		t.Errorf("string response should return nil, got: %v", err)
	}
	// Array body — definitively not an int.
	if err := c.waitIfJobResponse(context.Background(), []byte(`[]`), "test-op"); err != nil {
		t.Errorf("array response should return nil, got: %v", err)
	}
}

// TestDoOnce_TransportError exercises the early-return in doOnce when
// the HTTP client fails (unreachable server, DNS failure). The test
// points the client at a known-unused port on loopback — httpClient.Do
// returns an error immediately without reaching the wire.
func TestDoOnce_TransportError(t *testing.T) {
	c, err := New("http://127.0.0.1:1", "dummy") // port 1 is tcpmux, typically refused
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.RetryPolicy.MaxAttempts = 1 // don't retry; fail fast
	c.httpClient.Timeout = 0      // we want the connection refused, not a timeout

	_, err = c.Get(context.Background(), "/x")
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	// Must NOT be an APIError (that's only for HTTP status errors).
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("transport error should not be *APIError, got: %v", err)
	}
}
