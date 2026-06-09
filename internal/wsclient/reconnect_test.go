package wsclient

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestReconnectIfNeeded_alreadyHealthy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "ok", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Conn is alive after NewClient; reconnectIfNeeded should see
	// healthy and return nil without redialing.
	if err := c.reconnectIfNeeded(ctx); err != nil {
		t.Errorf("expected nil for healthy conn, got %v", err)
	}
}

func TestReconnectIfNeeded_clientClosed(t *testing.T) {
	c := &Client{
		closed:  make(chan struct{}),
		pending: make(map[uint64]chan *rpcResponse),
	}
	close(c.closed)
	err := c.reconnectIfNeeded(context.Background())
	if !errors.Is(err, ErrShuttingDown) {
		t.Errorf("expected ErrShuttingDown, got %v", err)
	}
}

func TestReconnectIfNeeded_redialSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "ok", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Force-drop the conn to simulate a transport failure.
	c.connMu.Lock()
	if c.conn != nil {
		// Mimic what a dropped conn would look like to recvLoop's
		// post-Read isClosing check, then nil out conn so reconnect
		// observes the unhealthy state.
		c.conn = nil
	}
	c.connMu.Unlock()

	if err := c.reconnectIfNeeded(ctx); err != nil {
		t.Errorf("expected redial to succeed, got %v", err)
	}

	c.connMu.Lock()
	healthy := c.conn != nil
	c.connMu.Unlock()
	if !healthy {
		t.Error("conn should be healthy after reconnect")
	}
}

func TestReconnectIfNeeded_dialFailure(t *testing.T) {
	// Client points at a URL that will never accept connections.
	c := &Client{
		baseURL:        "http://127.0.0.1:1",
		apiKey:         "k",
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		dialTimeout:    100 * time.Millisecond,
		requestTimeout: time.Second,
		RetryPolicy:    RetryPolicy{MaxAttempts: 2, BaseDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := c.reconnectIfNeeded(ctx)
	if err == nil {
		t.Fatal("expected redial failure, got nil")
	}
	// Either the deadline or a wrapped dial error is acceptable —
	// both prove the redial loop ran out without success.
}

func TestReconnectIfNeeded_ctxCanceled(t *testing.T) {
	c := &Client{
		baseURL:        "http://127.0.0.1:1",
		apiKey:         "k",
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		dialTimeout:    10 * time.Second,
		requestTimeout: time.Second,
		RetryPolicy:    RetryPolicy{MaxAttempts: 5, BaseDelay: 50 * time.Millisecond, MaxDelay: 200 * time.Millisecond},
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := c.reconnectIfNeeded(ctx)
	if err == nil {
		t.Fatal("expected ctx-canceled error, got nil")
	}
}

// TestReconnectIfNeeded_deadlineExceeded covers the time.Now().After
// (deadline) branch. dialTimeout = 50ms gives each dial enough time
// to return "connection refused" without surfacing as a deadline
// error; cumulative elapsed across 2-3 iterations crosses the 50ms
// deadline and exits via the deadline-after check.
func TestReconnectIfNeeded_deadlineExceeded(t *testing.T) {
	c := &Client{
		baseURL:        "http://127.0.0.1:1",
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		dialTimeout:    50 * time.Millisecond,
		requestTimeout: time.Second,
		// Short backoff so we cycle several times before the deadline.
		RetryPolicy: RetryPolicy{MaxAttempts: 10, BaseDelay: 5 * time.Millisecond, MaxDelay: 15 * time.Millisecond},
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.reconnectIfNeeded(ctx)
	if err == nil {
		t.Fatal("expected reconnect failure, got nil")
	}
	// Deadline-exceeded surfaces as ErrConnectionLost. ctx.DeadlineExceeded
	// is acceptable if the dial returns it (subsequent dials wrap into
	// the same error path either way).
	if !errors.Is(err, ErrConnectionLost) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected ErrConnectionLost or DeadlineExceeded, got %v", err)
	}
}

// TestReconnectIfNeeded_authFailure covers the dialErr = authErr path:
// dial succeeds against a live server but the server returns false to
// auth.login_with_api_key.
func TestReconnectIfNeeded_authFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "auth.login_with_api_key" {
			return false, nil
		}
		return nil, nil
	}, TestServerOptions{SkipAuth: true})

	// Build a Client that already considers itself unhealthy, then
	// point its baseURL at the auth-rejecting test server. reconnect
	// will redial successfully but then fail at the auth step.
	c := &Client{
		baseURL:        ts.URL(),
		apiKey:         "k",
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		dialTimeout:    1 * time.Second,
		requestTimeout: time.Second,
		RetryPolicy:    RetryPolicy{MaxAttempts: 1, BaseDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())

	err := c.reconnectIfNeeded(ctx)
	if err == nil {
		t.Fatal("expected reconnect to fail on auth, got nil")
	}
}

// TestReconnectIfNeeded_dialDeadlineExceeded covers the ctx-error
// early-return inside the reconnect loop. Server accepts TCP but
// blocks the upgrade so dialCtx times out → DeadlineExceeded.
func TestReconnectIfNeeded_dialDeadlineExceeded(t *testing.T) {
	srv := newRawHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Accept the upgrade slowly so dialCtx times out first.
		<-r.Context().Done()
	}))

	c := &Client{
		baseURL:        srv.URL,
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		dialTimeout:    20 * time.Millisecond,
		requestTimeout: time.Second,
		RetryPolicy:    RetryPolicy{MaxAttempts: 1, BaseDelay: 5 * time.Millisecond, MaxDelay: 10 * time.Millisecond},
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := c.reconnectIfNeeded(ctx)
	if err == nil {
		t.Fatal("expected reconnect failure on dial timeout")
	}
}

// TestReconnectIfNeeded_concurrentDoubleCheck covers the post-mutex
// double-check branch (line 53-55): goroutine A holds reconnectMu and
// reconnects; goroutine B parks on the mutex, then on entry sees the
// conn already healthy and returns nil without redialing.
func TestReconnectIfNeeded_concurrentDoubleCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "ok", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Force unhealthy.
	c.connMu.Lock()
	c.conn = nil
	c.connMu.Unlock()

	// Spawn two goroutines simultaneously. Whichever wins reconnectMu
	// reconnects; the other observes healthy conn after grabbing the
	// mutex and returns via the double-check branch.
	type res struct{ err error }
	resCh := make(chan res, 2)
	for i := 0; i < 2; i++ {
		go func() {
			resCh <- res{c.reconnectIfNeeded(ctx)}
		}()
	}

	for i := 0; i < 2; i++ {
		select {
		case r := <-resCh:
			if r.err != nil {
				t.Errorf("reconnect %d: unexpected err %v", i, r.err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("reconnect did not return within 5s")
		}
	}
}

func TestReconnectIfNeeded_badURL(t *testing.T) {
	c := &Client{
		baseURL:        "ftp://invalid-scheme",
		closed:         make(chan struct{}),
		pending:        make(map[uint64]chan *rpcResponse),
		dialTimeout:    100 * time.Millisecond,
		requestTimeout: time.Second,
	}
	err := c.reconnectIfNeeded(context.Background())
	if err == nil {
		t.Fatal("expected URL-parse error, got nil")
	}
}
