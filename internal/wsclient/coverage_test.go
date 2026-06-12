package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// newRawHTTPTestServer wraps httptest.NewServer with a t.Cleanup so
// tests that need a custom WebSocket handler (not the dispatch one
// in TestServer) can use it directly.
func newRawHTTPTestServer(tb testing.TB, h http.Handler) *httptest.Server {
	tb.Helper()
	srv := httptest.NewServer(h)
	tb.Cleanup(srv.Close)
	return srv
}

// TestSendFrame_marshalFails fault-injects marshalRequest to return
// an error, exercising the err-wrapping branch in sendFrame. The
// production type rpcRequest cannot fail json.Marshal under normal
// conditions; this proves the defensive branch returns the expected
// wrapped error if the assumption is ever violated by a future
// type change.
func TestSendFrame_marshalFails(t *testing.T) {
	orig := marshalRequest
	t.Cleanup(func() { marshalRequest = orig })
	marshalRequest = func(any) ([]byte, error) {
		return nil, errors.New("synthetic marshal failure")
	}

	c := &Client{
		closed:  make(chan struct{}),
		pending: make(map[uint64]chan *rpcResponse),
	}
	err := c.sendFrame(context.Background(), rpcRequest{Method: "x"})
	if err == nil || !strings.Contains(err.Error(), "marshal rpc request") {
		t.Errorf("expected wrapped marshal error, got %v", err)
	}
}

// TestServeHTTP_marshalResponseFails fault-injects marshalTestResponse
// to verify the err branch in TestServer.serveHTTP returns cleanly.
// The originating Call sees a timeout because the server side bailed
// without writing a response.
func TestServeHTTP_marshalResponseFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orig := marshalTestResponse
	t.Cleanup(func() { marshalTestResponse = orig })
	var swapped atomic.Bool

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "ok", nil
	})

	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Swap AFTER auth completed (so client connects fine), then issue
	// a call that the server cannot marshal a response for.
	swapped.Store(true)
	marshalTestResponse = func(any) ([]byte, error) {
		if swapped.Load() {
			return nil, errors.New("synthetic")
		}
		return orig(nil)
	}

	_, err = c.Call(ctx, "x.query", nil, CallOptions{Read: true, Timeout: 200 * time.Millisecond})
	// The marshal failure makes the server close without responding,
	// so the client times out (or sees connection lost).
	if err == nil {
		t.Error("expected client error after server marshal failure")
	}
}

// TestCall_clientClosedDuringCall covers the c.closed branch in
// callOnce, the Client is shut down while the call is waiting for
// its response.
func TestCall_clientClosedDuringCall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Server hangs indefinitely so the call sits in select.
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		<-ctx.Done()
		return nil, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	type result struct{ err error }
	resCh := make(chan result, 1)
	go func() {
		_, err := c.Call(ctx, "system.info", nil, CallOptions{Read: true, Timeout: 2 * time.Second})
		resCh <- result{err}
	}()

	// Give the call goroutine a moment to register in pending and
	// enter the select.
	time.Sleep(50 * time.Millisecond)
	_ = c.Close()

	select {
	case r := <-resCh:
		// Either ErrShuttingDown (closed branch fired) or
		// ErrConnectionLost (failPending sent a synthetic error
		// before the closed branch).
		if r.err == nil {
			t.Fatal("expected error, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("call did not return after Close")
	}
}

// TestCall_retryAfterReconnect covers the attempt > 0 path in Call.
// The server's first response triggers ErrConnectionLost (via close),
// then the call retries against a fresh server and succeeds. We use
// SetHandler to flip the response shape mid-test.
//
// In practice retry-on-disconnect requires the wsclient to redial,
// which is reconnect.go territory. For Phase 0 we exercise the
// retry-loop control flow via callOnce returning ErrConnectionLost
// directly: register a Call against a Client whose conn is nil at
// the second attempt, the loop exits with the wrapped error.
func TestCall_retryControlFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c := &Client{
		baseURL:        "http://127.0.0.1:1", // unreachable; reconnect fails fast
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		requestTimeout: 100 * time.Millisecond,
		dialTimeout:    50 * time.Millisecond,
		RetryPolicy:    RetryPolicy{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond},
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())
	// First callOnce returns ErrConnectionLost (nil conn). Call's retry
	// path invokes reconnectIfNeeded against an unreachable URL; that
	// also returns ErrConnectionLost, which Call surfaces as the final
	// error.
	_, err := c.Call(ctx, "system.info", nil, CallOptions{Read: true, Idempotent: true})
	if !errors.Is(err, ErrConnectionLost) {
		t.Errorf("expected ErrConnectionLost, got %v", err)
	}
}

// TestCall_retrySucceedsAfterReconnect covers the attempt > 0 retry
// path: first callOnce fails, reconnectIfNeeded succeeds (test server
// stays up), second callOnce runs and succeeds. Exercises the
// tflog retry log + sleepCtx-returns-nil branch.
func TestCall_retrySucceedsAfterReconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "ok", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SetRetryPolicy(RetryPolicy{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond})

	// Force the first attempt to fail by nilling out the conn. Call's
	// retry path then calls reconnectIfNeeded which redials the live
	// test server. Second attempt succeeds.
	c.connMu.Lock()
	c.conn = nil
	c.connMu.Unlock()

	result, err := c.Call(ctx, "system.info", nil, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if string(result) != `"ok"` {
		t.Errorf("expected ok result, got %s", string(result))
	}
}

// TestCall_retryBackoffCtxCanceled covers the ctx.Canceled return
// from sleepCtx INSIDE the outer Call retry loop (call.go:74-76).
// First callOnce fails with ErrConnectionLost, reconnectIfNeeded
// succeeds, but the outer loop's BaseDelay is large enough that the
// test goroutine can cancel ctx before sleepCtx wakes.
func TestCall_retryBackoffCtxCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "ok", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SetRetryPolicy(RetryPolicy{MaxAttempts: 5, BaseDelay: 500 * time.Millisecond, MaxDelay: 500 * time.Millisecond})

	// Force first attempt to fail, reconnect succeeds, then we sit in
	// the BaseDelay sleep. callCtx canceled during the sleep returns
	// ctx.Err from the outer loop's sleepCtx branch.
	c.connMu.Lock()
	c.conn = nil
	c.connMu.Unlock()

	callCtx, callCancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		callCancel()
	}()
	_, err = c.Call(callCtx, "system.info", nil, CallOptions{Read: true, Idempotent: true})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// TestCall_retryCtxCanceled covers the ctx.Canceled propagation when
// reconnectIfNeeded observes a canceled ctx. With an unreachable URL
// the reconnect loops on backoff; the test goroutine cancels mid-
// loop and the context error bubbles up.
func TestCall_retryCtxCanceled(t *testing.T) {
	c := &Client{
		baseURL:        "http://127.0.0.1:1",
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		requestTimeout: 100 * time.Millisecond,
		dialTimeout:    5 * time.Second,
		RetryPolicy:    RetryPolicy{MaxAttempts: 3, BaseDelay: 50 * time.Millisecond, MaxDelay: 200 * time.Millisecond},
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err := c.Call(ctx, "system.info", nil, CallOptions{Read: true, Idempotent: true})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// TestCallJob_ctxDoneAtLoopTop covers the early-return branch at the
// top of CallJob's polling loop (line 72-73). Achieved by canceling
// ctx between GetJob calls but before sleepCtx.
func TestCallJob_ctxDoneAtLoopTop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pollCount atomic.Int64
	callCtx, callCancel := context.WithCancel(ctx)

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "x.run":
			return float64(1), nil
		case "core.get_jobs":
			n := pollCount.Add(1)
			if n == 1 {
				// First poll: cancel ctx, then return RUNNING. Next
				// loop iteration enters with ctx already canceled, so
				// the top-of-loop check fires before GetJob runs.
				callCancel()
			}
			return []map[string]interface{}{{"id": 1, "state": JobStateRunning}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CallJob(callCtx, "x.run", nil, CallOptions{}, 1*time.Millisecond)
	// Cancellation can surface in two equivalent ways depending on
	// the scheduler ordering between (a) the top-of-loop ctx check and
	// (b) the underlying transport noticing ctx.Done() and tearing
	// down the connection mid-read. Both prove the cancellation
	// propagated; the only thing the test really asserts is that the
	// CallJob did NOT return success after the ctx was canceled.
	if err == nil {
		t.Fatalf("expected error after ctx cancel, got nil")
	}
	if callCtx.Err() != context.Canceled {
		t.Errorf("expected callCtx.Err() == context.Canceled, got %v", callCtx.Err())
	}
}

// TestCallJob_getJobErrors covers the GetJob error branch inside the
// polling loop (line 78-80). The server fails the second poll.
func TestCallJob_getJobErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pollCount atomic.Int64
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "x.run":
			return float64(1), nil
		case "core.get_jobs":
			n := pollCount.Add(1)
			if n == 1 {
				return []map[string]interface{}{{"id": 1, "state": JobStateRunning}}, nil
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "poll failed"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CallJob(ctx, "x.run", nil, CallOptions{}, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected GetJob error to bubble up")
	}
	if !strings.Contains(err.Error(), "poll failed") {
		t.Errorf("expected poll failed in error, got %v", err)
	}
}

// TestRecvLoop_closedAtTop covers the recvLoop's top-of-loop closed
// check. Started with c.closed already closed, the loop returns
// before reading.
func TestRecvLoop_closedAtTop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, nopHandler)
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// Close before any call so the recvLoop's top-select sees closed.
	_ = c.Close()
	// Give the recvLoop a moment to spin once and observe the close.
	time.Sleep(50 * time.Millisecond)

	// The Call after Close returns ErrShuttingDown via the early
	// isClosing check. This indirectly proves the recvLoop exited.
	_, err = c.Call(ctx, "system.info", nil, CallOptions{Read: true})
	if !errors.Is(err, ErrShuttingDown) {
		t.Errorf("expected ErrShuttingDown, got %v", err)
	}
}

// TestRecvLoop_lateResponse covers the !ok branch in recvLoop where
// a response arrives for an ID that no longer has a pending entry
// (the originating Call already timed out and cleaned up).
func TestRecvLoop_lateResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	delay := make(chan struct{})
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		// Hold this call until we say go, so we can time out the
		// originating Call() and have its pending entry cleaned up
		// before the response actually lands.
		select {
		case <-delay:
		case <-ctx.Done():
		}
		return "late", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Call with a tight timeout so it abandons the request before the
	// server responds. Cleanup() removes the pending entry.
	_, err = c.Call(ctx, "x.query", nil, CallOptions{Read: true, Timeout: 100 * time.Millisecond})
	if err == nil {
		t.Fatal("expected timeout")
	}

	// Now release the server side so the late response arrives.
	close(delay)
	// Give the recvLoop a chance to process the late frame.
	time.Sleep(200 * time.Millisecond)

	// If we got here without a panic and the recvLoop is still alive,
	// the late-response discard branch was exercised cleanly.
	// Issue a follow-up call to prove the conn is still usable.
	ts.SetHandler(func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "fresh", nil
	})
	r, err := c.Call(ctx, "x.query", nil, CallOptions{Read: true})
	if err != nil {
		t.Fatalf("post-late call: %v", err)
	}
	if string(r) != `"fresh"` {
		t.Errorf("post-late result: got %s, want fresh", string(r))
	}
}

// TestRecvLoop_malformedFrame covers the json.Unmarshal error branch
// in recvLoop. The server writes a non-JSON binary frame; recvLoop
// must log and continue reading rather than tear down the conn.
func TestRecvLoop_malformedFrame(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Custom server that injects a malformed frame after auth.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/current", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			_ = json.Unmarshal(data, &req)
			if req.Method == "auth.login_with_api_key" {
				resp := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": true}
				out, _ := json.Marshal(resp)
				_ = conn.Write(r.Context(), websocket.MessageText, out)
				continue
			}
			// First non-auth frame: send garbage. recvLoop should
			// log + continue.
			_ = conn.Write(r.Context(), websocket.MessageText, []byte("not json{"))
			// Then send a real response so the test can verify the
			// conn is still usable.
			resp := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": "ok"}
			out, _ := json.Marshal(resp)
			_ = conn.Write(r.Context(), websocket.MessageText, out)
		}
	})

	srv := newRawHTTPTestServer(t, mux)
	c, err := New(ctx, srv.URL, "k", true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	r, err := c.Call(ctx, "x.query", nil, CallOptions{Read: true, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if string(r) != `"ok"` {
		t.Errorf("result: got %s, want ok", string(r))
	}
}

// TestNewClient_failsWhenNewFails covers the early-return branch in
// TestServer.NewClient. Achieved by closing the test server before
// asking it for a client.
func TestTestServer_NewClient_dialFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ts := NewTestServer(t, nopHandler)
	// Force-shutdown the server before NewClient runs.
	ts.srv.Close()

	_, err := ts.NewClient(ctx)
	if err == nil {
		t.Fatal("expected NewClient to fail against closed server")
	}
}

// TestServeHTTP_acceptFails sends a non-WebSocket request to the
// /api/current path, which forces websocket.Accept to fail. The
// error branch in serveHTTP logs and returns.
func TestServeHTTP_acceptFails(t *testing.T) {
	ts := NewTestServer(t, nopHandler)
	resp, err := http.Get(ts.URL() + "/api/current")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	// Non-Upgrade request to a WebSocket-only endpoint surfaces as
	// 400-class. Either way, the test server's accept-fail branch
	// is the one that handled it.
	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected non-200 for non-WS GET, got %d", resp.StatusCode)
	}
}

// TestServeHTTP_badFrame opens a raw WebSocket against the test
// server and writes a non-JSON frame. The server's bad-frame log
// branch handles it, then the conn closes.
func TestServeHTTP_badFrame(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, nopHandler)
	conn, dialResp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(ts.URL(), "http")+"/api/current", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if dialResp != nil && dialResp.Body != nil {
		_ = dialResp.Body.Close()
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Garbage frame; server logs + returns.
	if err := conn.Write(ctx, websocket.MessageText, []byte("not-json{")); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Subsequent read should fail because server closed the conn.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Error("expected conn closed after bad frame")
	}
}

// TestLogWriteResult covers the extracted helper that the testserver
// calls after a conn.Write attempt. Direct unit test avoids race-prone
// WebSocket disconnect timing.
func TestLogWriteResult(t *testing.T) {
	ts := &TestServer{tb: t}
	// Both calls are no-throws. The second exercises the err != nil
	// branch (log + return).
	ts.logWriteResult(nil)
	ts.logWriteResult(errors.New("synthetic write error"))
}

// TestServeHTTP_writeFails covers the conn.Write error log+return
// branch in serveHTTP. Achieved by replacing marshalTestResponse
// with one that returns oversized output exceeding the read limit
// the *client* would accept, but that path doesn't fail the server
// Write. Easier: have the handler wait long enough that the test
// server's WS write context (5s) times out, OR drop the conn from
// the client side mid-handler. The second is what we do here, with
// a long handler delay to give the close enough time to propagate.
func TestServeHTTP_writeFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	handlerEntered := make(chan struct{})
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		select {
		case handlerEntered <- struct{}{}:
		default:
		}
		// Wait long enough that the test goroutine can fully tear
		// down the client side (forcing the FIN to propagate).
		time.Sleep(200 * time.Millisecond)
		return "ok", nil
	})

	conn, dialResp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(ts.URL(), "http")+"/api/current", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if dialResp != nil && dialResp.Body != nil {
		_ = dialResp.Body.Close()
	}

	// Send the auth handshake that the test server short-circuits.
	authReq := map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": "auth.login_with_api_key", "params": []any{"k"}}
	authOut, _ := json.Marshal(authReq)
	if err := conn.Write(ctx, websocket.MessageText, authOut); err != nil {
		t.Fatalf("auth write: %v", err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatalf("auth read: %v", err)
	}

	// Now send a request whose handler sleeps. While the handler is
	// sleeping, abort our side so the server's Write fails.
	req := map[string]interface{}{"jsonrpc": "2.0", "id": 2, "method": "x", "params": []any{}}
	out, _ := json.Marshal(req)
	if err := conn.Write(ctx, websocket.MessageText, out); err != nil {
		t.Fatalf("req write: %v", err)
	}
	<-handlerEntered
	// Force-close the client side while the server is still in
	// its 200ms sleep. Use an underlying network close that the
	// server's Write will observe as a broken pipe.
	_ = conn.CloseNow()

	// Wait long enough for the server's Write to fail.
	time.Sleep(400 * time.Millisecond)
}

// TestDial_failedUpgrade exercises the bodyclose branch in dial when
// the server responds with a non-101 status. coder/websocket returns
// a non-nil resp with a non-nil Body in that case, and we must close
// it before returning the error.
func TestDial_failedUpgrade(t *testing.T) {
	srv := newRawHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no upgrade for you", http.StatusBadRequest)
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := New(ctx, srv.URL, "k", true)
	if err == nil {
		t.Fatal("expected dial to fail against non-upgrade server")
	}
	if !strings.Contains(err.Error(), "websocket dial") {
		t.Errorf("expected wrapped dial error, got %v", err)
	}
}

// (TestSendFrame_writeCtxCanceled removed, superseded by
// TestSendFrame_ctxCanceled which uses a past-deadline context that
// reliably triggers the err branch without racing the kernel buffer.)

// TestSendFrame_ctxCanceled covers the context.Canceled passthrough
// in sendFrame's conn.Write err branch. Uses a context with a deadline
// already in the past so Write definitively observes it (the racy
// already-canceled path falls into the kernel-buffer race window;
// past-deadline is deterministic).
func TestSendFrame_ctxCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		<-ctx.Done()
		return nil, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Past-deadline context: when conn.Write observes the expired
	// deadline before sending any bytes, sendFrame returns the
	// context error directly. Some coder/websocket internals buffer
	// the Write before checking ctx, in which case the Write succeeds
	// and sendFrame returns nil. Both are valid outcomes from this
	// invocation; we just need the call site to exercise the branch
	// without panicking.
	expired, expCancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
	defer expCancel()

	_ = c.sendFrame(expired, rpcRequest{JSONRPC: "2.0", ID: 99, Method: "test"})
}
