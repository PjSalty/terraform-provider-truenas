package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestHandler is the per-call mock that test code provides to a
// TestServer. It receives the JSON-RPC method name and decoded
// params, and returns either a result (any value json.Marshal can
// serialize) or an *RPCError. Both nil result and nil error means
// "return null result", which matches a few TrueNAS methods that
// signal success via no payload.
type TestHandler func(ctx context.Context, method string, params []interface{}) (result interface{}, rpcErr *RPCError)

// TestServer is a wraps an httptest.Server running a WebSocket upgrade
// handler that dispatches JSON-RPC frames to a user-supplied handler.
// Callers get a *Client pre-connected to the server via NewClient().
//
// Mirrors the REST client's newTestServer pattern so the test rewrite
// cost during migration is "swap helper, keep assertions."
//
// One TestServer should outlive its t.Cleanup; the server is shut down
// automatically when the test ends.
type TestServer struct {
	srv         *httptest.Server
	handler     TestHandler
	tb          testing.TB
	mu          sync.Mutex
	skipAuth    bool
	activeConns []*websocket.Conn
}

// NewTestServer starts an httptest.Server on a random port that
// upgrades to WebSocket and dispatches every incoming JSON-RPC frame
// through h. Calls t.Cleanup to shut down the server when the test
// finishes.
//
// The server intercepts auth.login_with_api_key automatically and
// returns true; tests that need to exercise auth failure can pass the
// SkipAuth option (see NewTestServerWithOptions) and provide their own
// handler.
func NewTestServer(tb testing.TB, h TestHandler) *TestServer {
	tb.Helper()
	return NewTestServerWithOptions(tb, h, TestServerOptions{})
}

// TestServerOptions configures NewTestServerWithOptions.
type TestServerOptions struct {
	// SkipAuth, when true, suppresses the built-in auth.login_with_api_key
	// short-circuit. The user-supplied handler must field the auth call
	// itself.
	SkipAuth bool
}

// NewTestServerWithOptions is the configurable form of NewTestServer.
func NewTestServerWithOptions(tb testing.TB, h TestHandler, opts TestServerOptions) *TestServer {
	tb.Helper()
	ts := &TestServer{handler: h, tb: tb, skipAuth: opts.SkipAuth}
	ts.srv = httptest.NewServer(http.HandlerFunc(ts.serveHTTP))
	tb.Cleanup(func() { ts.srv.Close() })
	return ts
}

// URL returns the http:// URL of the server. NewClient() converts this
// to ws:// internally.
func (ts *TestServer) URL() string { return ts.srv.URL }

// NewClient dials the test server and returns a connected, authenticated
// Client. The client is registered for cleanup so callers do not need
// to defer Close().
func (ts *TestServer) NewClient(ctx context.Context) (*Client, error) {
	c, err := New(ctx, ts.srv.URL, "test-api-key", true)
	if err != nil {
		return nil, err
	}
	ts.tb.Cleanup(func() { _ = c.Close() })
	return c, nil
}

// serveHTTP handles the upgrade and the per-frame dispatch loop.
func (ts *TestServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	// Sanity: TrueNAS pins the WebSocket path to /api/current. Mirror
	// that so a test client misconfigured against /api/v2.0 fails fast.
	if !strings.HasSuffix(r.URL.Path, "/api/current") {
		http.NotFound(w, r)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		ts.tb.Logf("test server accept: %v", err)
		return
	}
	ts.mu.Lock()
	ts.activeConns = append(ts.activeConns, conn)
	ts.mu.Unlock()
	// nhooyr's default close on handler exit is StatusInternalError
	// which makes for noisy logs in normal teardown. Wrap to convert
	// "handler returned" into a normal close.
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "test server done") }()

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      uint64          `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			ts.tb.Logf("test server: bad request frame: %v", err)
			return
		}

		var params []interface{}
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &params)
		}

		var (
			result interface{}
			rpcErr *RPCError
		)
		if !ts.skipAuth && req.Method == "auth.login_with_api_key" {
			result = true
		} else {
			ts.mu.Lock()
			h := ts.handler
			ts.mu.Unlock()
			if h != nil {
				result, rpcErr = h(r.Context(), req.Method, params)
			} else {
				rpcErr = &RPCError{Code: CodeMethodNotFound, Message: "no handler installed: " + req.Method}
			}
		}

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
		}
		if rpcErr != nil {
			resp["error"] = rpcErr
		} else {
			resp["result"] = result
		}

		out, err := marshalTestResponse(resp)
		if err != nil {
			// Truly unreachable in normal use, resp is a well-typed
			// map with json-friendly values. Log and bail rather than
			// tb.Fatalf so a fault-injection test that swaps
			// marshalTestResponse can exercise this branch without
			// taking down the surrounding test.
			ts.tb.Logf("test server: marshal response failed (fault injected?): %v", err)
			return
		}
		writeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		writeErr := conn.Write(writeCtx, websocket.MessageText, out)
		cancel()
		// On write failure the conn is hosed; the next conn.Read at
		// the top of the loop will return an error and exit the
		// goroutine cleanly. Log diagnostics here, otherwise let the
		// loop progress so we don't add an untestable branch.
		ts.logWriteResult(writeErr)
	}
}

// logWriteResult logs a write failure if there was one. Extracted to
// keep serveHTTP branchless on the rare error case, testing a real
// mid-write WebSocket disconnect is race-prone, but a direct call to
// this helper covers both the err and no-err paths trivially.
func (ts *TestServer) logWriteResult(err error) {
	if err != nil {
		ts.tb.Logf("test server: write response: %v", err)
	}
}

// SetHandler swaps the active handler at runtime. Useful in tests that
// drive the same connection through multiple phases (e.g. first reply
// 500, then reply 200).
func (ts *TestServer) SetHandler(h TestHandler) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.handler = h
}

// CloseConnection abruptly closes every active WebSocket connection
// the server has accepted, simulating a transport drop. Subsequent
// client calls return ErrConnectionLost (or time out, depending on
// whether the kernel has flushed the FIN by the time the call's
// sendFrame runs). Used to exercise reconnect logic.
//
// The httptest.Server itself remains alive, only the WS conns are
// dropped, so reconnect tests can issue a follow-up dial against
// the same URL.
func (ts *TestServer) CloseConnection() {
	ts.mu.Lock()
	conns := ts.activeConns
	ts.activeConns = nil
	ts.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close(websocket.StatusGoingAway, "test forced disconnect")
	}
}

// marshalTestResponse is the JSON marshaler for outgoing test-server
// responses. Defaults to json.Marshal; covered by the inline
// fault-injection test that swaps it for a failing impl.
var marshalTestResponse = json.Marshal

// nopHandler is a sentinel that returns CodeMethodNotFound for every
// call. Useful for client tests that exercise error paths without
// caring about per-method behavior.
func nopHandler(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
	_ = ctx
	_ = params
	return nil, &RPCError{Code: CodeMethodNotFound, Message: "nopHandler: " + method}
}

// ErrTestServerClosed is returned from a few helpers when the test
// server has been torn down before the helper completes.
var ErrTestServerClosed = errors.New("test server closed")
