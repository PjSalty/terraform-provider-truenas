package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDestroyProtection_BlocksDELETE asserts that when the flag is on,
// a DELETE request fails with ErrDestroyProtected BEFORE the network
// call is made. The httptest server counts hits so we can prove the
// target system never observed the request.
func TestDestroyProtection_BlocksDELETE(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.DestroyProtection = true

	_, err = c.Delete(context.Background(), "/dataset/id/test%2Fdangerous")
	if err == nil {
		t.Fatal("expected DELETE to be refused, got nil error")
	}
	if !errors.Is(err, ErrDestroyProtected) {
		t.Errorf("expected ErrDestroyProtected, got %v", err)
	}
	if hits != 0 {
		t.Errorf("server received %d hits — client must refuse BEFORE network", hits)
	}
}

// TestDestroyProtection_AllowsCreateUpdateRead asserts that when the
// flag is on, POST/PUT/GET all flow through normally. This is the
// "safe apply" profile: first production apply can create and update
// resources freely while destroys are blocked at the wire.
func TestDestroyProtection_AllowsCreateUpdateRead(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.DestroyProtection = true

	if _, err := c.Get(context.Background(), "/dataset"); err != nil {
		t.Errorf("GET refused: %v", err)
	}
	if _, err := c.Post(context.Background(), "/dataset", map[string]string{"name": "test"}); err != nil {
		t.Errorf("POST refused: %v", err)
	}
	if _, err := c.Put(context.Background(), "/dataset/id/test", map[string]string{"name": "test"}); err != nil {
		t.Errorf("PUT refused: %v", err)
	}
	if hits != 3 {
		t.Errorf("expected 3 hits (GET/POST/PUT), got %d", hits)
	}
}

// TestDestroyProtection_Disabled is the "off by default" check: when
// DestroyProtection is false, DELETE flows through just like any other
// verb. This is the default provider behavior so existing users aren't
// silently affected by the new flag.
func TestDestroyProtection_Disabled(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// DestroyProtection defaults to false.

	if _, err := c.Delete(context.Background(), "/dataset/id/test"); err != nil {
		t.Errorf("DELETE refused with flag off: %v", err)
	}
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
}

// TestDestroyProtection_LayeredWithReadOnly asserts that ReadOnly
// strictly dominates DestroyProtection: when both are set, a POST is
// refused under ErrReadOnly (the broader error) rather than silently
// passing through the destroy-protection check. This matters because
// operators who set both flags expect the tightest possible gate.
func TestDestroyProtection_LayeredWithReadOnly(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.ReadOnly = true
	c.DestroyProtection = true

	// POST must be blocked by ReadOnly (DestroyProtection alone would not).
	_, err = c.Post(context.Background(), "/dataset", map[string]string{})
	if !errors.Is(err, ErrReadOnly) {
		t.Errorf("expected ErrReadOnly for POST under layered flags, got %v", err)
	}
	// DELETE must also be blocked — by ReadOnly first (dominates).
	_, err = c.Delete(context.Background(), "/dataset/id/x")
	if !errors.Is(err, ErrReadOnly) {
		t.Errorf("expected ErrReadOnly for DELETE under layered flags, got %v", err)
	}
	if hits != 0 {
		t.Errorf("no requests should reach server, got %d hits", hits)
	}
}

// TestDestroyProtection_NilClient is defensive: a nil client receiver
// must not panic when the safety check is called. Unlikely in practice
// but matches the symmetric guard in checkReadOnly.
func TestDestroyProtection_NilClient(t *testing.T) {
	var c *Client
	if err := c.checkDestroyProtection("DELETE", "/x"); err != nil {
		t.Errorf("nil client should return nil, got %v", err)
	}
}

// TestDestroyProtection_ErrorWraps checks that errors.Is works, so
// callers can write `if errors.Is(err, ErrDestroyProtected)` rather
// than string-matching. This is the same contract as ErrReadOnly.
func TestDestroyProtection_ErrorWraps(t *testing.T) {
	c := &Client{DestroyProtection: true}
	err := c.checkDestroyProtection("DELETE", "/anywhere")
	if !errors.Is(err, ErrDestroyProtected) {
		t.Errorf("errors.Is(err, ErrDestroyProtected) = false, got err=%v", err)
	}
	if err.Error() == "" {
		t.Error("wrapped error must have a non-empty message")
	}
}
