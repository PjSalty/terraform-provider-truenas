package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDoOnceRedactsAPIErrorBody is an end-to-end wiring test: stand up
// an httptest server that returns a 422 carrying a sensitive field in
// its JSON body, drive doRequest through it, and assert that the
// APIError bubbled back has NO trace of the secret value.
//
// This is the Phase G ratchet: if a future refactor accidentally
// bypasses redactJSONBody on the error path, this test fails loudly.
func TestDoOnceRedactsAPIErrorBody(t *testing.T) {
	const leakMarker = "LEAK-MARKER-SHOULD-NEVER-APPEAR-IN-DIAGNOSTICS"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "validation failed",
			"trace": map[string]interface{}{
				"config": map[string]interface{}{
					"password":   leakMarker,
					"dhchap_key": leakMarker,
					"api_key":    leakMarker,
				},
			},
		})
	}))
	defer srv.Close()

	c, err := New(srv.URL, "dummy-api-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Disable retries so we don't spam the test server and so the test
	// deterministically surfaces the first error.
	c.RetryPolicy.MaxAttempts = 1

	_, err = c.Get(context.Background(), "/some/path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Assertion 1: err.Error() — which feeds directly into
	// resp.Diagnostics.AddError on every resource CRUD path — must not
	// leak the secret value under any code path.
	msg := err.Error()
	if strings.Contains(msg, leakMarker) {
		t.Fatalf("LEAK: error message contains secret marker %q: %s", leakMarker, msg)
	}

	// Assertion 2: the underlying APIError.Body must also be redacted,
	// so that future refactors that e.g. extend Error() to include body
	// details for deeper diagnostics remain safe by construction.
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if strings.Contains(apiErr.Body, leakMarker) {
		t.Fatalf("LEAK: APIError.Body contains secret marker: %s", apiErr.Body)
	}
	if !strings.Contains(apiErr.Body, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] marker in APIError.Body, got: %s", apiErr.Body)
	}
}

// TestDoOnceRedactsMessageField specifically covers the Message branch
// (parsed `message` field from the JSON response). If the server
// describes the offending field by name, the redactMessage pass must
// truncate before the sensitive fragment appears.
func TestDoOnceRedactsMessageField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"message":"validation failed: password is too weak for user"}`)
	}))
	defer srv.Close()

	c, err := New(srv.URL, "dummy")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.RetryPolicy.MaxAttempts = 1

	_, err = c.Get(context.Background(), "/x")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	// After redactMessage, the word "password" marks the truncation point.
	// Everything from "password" onward should be replaced with [REDACTED].
	if strings.Contains(msg, "is too weak for user") {
		t.Fatalf("message was not truncated at sensitive fragment: %s", msg)
	}
	if !strings.Contains(msg, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] marker: %s", msg)
	}
}
