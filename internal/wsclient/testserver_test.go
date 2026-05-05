package wsclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTestServer_NotFoundOnWrongPath(t *testing.T) {
	ts := NewTestServer(t, nopHandler)
	resp, err := http.Get(ts.URL() + "/api/v2.0/wrong")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-/api/current path, got %d", resp.StatusCode)
	}
}

func TestTestServer_NopHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// nopHandler always returns CodeMethodNotFound.
	ts := NewTestServer(t, nopHandler)
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.Call(ctx, "anything", nil, CallOptions{Read: true})
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) || rpcErr.Code != CodeMethodNotFound {
		t.Errorf("expected MethodNotFound, got %v", err)
	}
}

func TestTestServer_SetHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	first := func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "first", nil
	}
	second := func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "second", nil
	}

	ts := NewTestServer(t, first)
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	r1, err := c.Call(ctx, "x.query", nil, CallOptions{Read: true})
	if err != nil {
		t.Fatalf("Call 1: %v", err)
	}
	if string(r1) != `"first"` {
		t.Errorf("got %s, want first", string(r1))
	}

	ts.SetHandler(second)
	r2, err := c.Call(ctx, "x.query", nil, CallOptions{Read: true})
	if err != nil {
		t.Fatalf("Call 2: %v", err)
	}
	if string(r2) != `"second"` {
		t.Errorf("got %s, want second", string(r2))
	}
}

func TestTestServer_CloseConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "ok", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// First call works.
	if _, err := c.Call(ctx, "x.query", nil, CallOptions{Read: true}); err != nil {
		t.Fatalf("Call 1: %v", err)
	}

	// Force the server to drop the connection.
	ts.CloseConnection()

	// Give the receive loop a moment to notice the conn drop and
	// drain pending callers via failPending. The actual TCP FIN may
	// take a moment to propagate through nhooyr's read loop.
	deadline := time.After(2 * time.Second)
	for {
		_, err = c.Call(ctx, "x.query", nil, CallOptions{Read: true, Timeout: 200 * time.Millisecond})
		if err != nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("expected error after CloseConnection within 2s, calls keep succeeding")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func TestTestServer_ErrSentinel(t *testing.T) {
	if ErrTestServerClosed.Error() == "" {
		t.Error("ErrTestServerClosed has empty message")
	}
}

func TestTestServer_NoHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// nil handler causes the server to return MethodNotFound for any
	// method that is not the auth handshake. Verifies the defensive
	// h==nil branch in serveHTTP.
	ts := &TestServer{tb: t}
	srv := httptest.NewServer(http.HandlerFunc(ts.serveHTTP))
	t.Cleanup(srv.Close)
	ts.srv = srv

	c, err := New(ctx, srv.URL, "k", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	_, err = c.Call(ctx, "anything.else", nil, CallOptions{Read: true})
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) || rpcErr.Code != CodeMethodNotFound {
		t.Errorf("expected MethodNotFound, got %v", err)
	}
	if !strings.Contains(rpcErr.Message, "no handler installed") {
		t.Errorf("expected 'no handler installed' message, got %q", rpcErr.Message)
	}
}
