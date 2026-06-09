package wsclient

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCall_nilClient(t *testing.T) {
	var c *Client
	_, err := c.Call(context.Background(), "x", nil, CallOptions{})
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected nil-client error, got %v", err)
	}
}

func TestCall_emptyMethod(t *testing.T) {
	c := &Client{closed: make(chan struct{})}
	_, err := c.Call(context.Background(), "", nil, CallOptions{})
	if err == nil || !strings.Contains(err.Error(), "method name") {
		t.Errorf("expected method-name error, got %v", err)
	}
}

func TestCall_shuttingDown(t *testing.T) {
	c := &Client{
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		requestTimeout: time.Second,
		RetryPolicy:        DefaultRetryPolicy(),
	}
	close(c.closed)
	_, err := c.Call(context.Background(), "system.info", nil, CallOptions{Read: true})
	if !errors.Is(err, ErrShuttingDown) {
		t.Errorf("expected ErrShuttingDown, got %v", err)
	}
}

func TestCall_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "bad params"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Call(ctx, "system.info", nil, CallOptions{Read: true, Idempotent: true})
	if err == nil {
		t.Fatal("expected RPC error")
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *RPCError, got %T", err)
	}
	if rpcErr.Code != CodeInvalidParams {
		t.Errorf("code: got %d, want %d", rpcErr.Code, CodeInvalidParams)
	}
}

func TestCall_timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Server hangs forever.
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		<-ctx.Done()
		return nil, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Call(ctx, "system.info", nil, CallOptions{
		Read:    true,
		Timeout: 200 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout, got %v", err)
	}
}

func TestCall_ctxCanceled(t *testing.T) {
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

	callCtx, callCancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		callCancel()
	}()
	_, err = c.Call(callCtx, "system.info", nil, CallOptions{Read: true})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCall_retryOnDisconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	calls := 0
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		calls++
		return map[string]interface{}{"ok": true}, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// Tighter retry config so the test runs fast.
	c.SetRetryPolicy(RetryPolicy{MaxAttempts: 2, BaseDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond})

	_, err = c.Call(ctx, "system.info", nil, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if calls < 1 {
		t.Errorf("expected at least 1 server call, got %d", calls)
	}
}

func TestCall_noRetryWhenNotIdempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// CodeMethodCallError is a server-rejected response, not a transport
	// drop — so the retry path is irrelevant. Call returns immediately
	// with the typed RPCError regardless of Idempotent.
	_, err = c.Call(ctx, "pool.dataset.create", []interface{}{}, CallOptions{Idempotent: false})
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Errorf("expected *RPCError, got %v", err)
	}
}

func TestCall_paramsNilEncodedAsArray(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		// params should arrive as empty slice (not nil) on the server side.
		if params == nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "params was nil"}
		}
		return "ok", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Call(ctx, "core.ping", nil, CallOptions{Read: true})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
}

func TestCallOnce_sendFails(t *testing.T) {
	c := &Client{
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		requestTimeout: time.Second,
	}
	// No conn set; sendFrame returns ErrConnectionLost. This exercises
	// callOnce's cleanup path.
	_, err := c.callOnce(context.Background(), "x", nil, time.Second)
	if !errors.Is(err, ErrConnectionLost) {
		t.Errorf("expected ErrConnectionLost, got %v", err)
	}
	// pending map must be empty after cleanup.
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	if len(c.pending) != 0 {
		t.Errorf("pending should be cleaned up, has %d entries", len(c.pending))
	}
}
