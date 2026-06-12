package wsclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestAuthenticate_RetriesPastRateLimit verifies the auth retry loop
// recovers when TrueNAS returns [EBUSY] Rate Limit Exceeded for the
// first N attempts and succeeds on the (N+1)th. Bookkeeping is
// per-connection: the test server tracks the auth attempt count and
// flips the response shape when the threshold clears.
func TestAuthenticate_RetriesPastRateLimit(t *testing.T) {
	const failFirst = 3
	var attempts atomic.Int32

	// Run our own minimal JSON-RPC server because the existing
	// NewTestServer short-circuits auth before counting attempts.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer conn.CloseNow()

		// Receive auth call frames and respond with EBUSY for
		// failFirst calls, then success.
		for {
			_, msg, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var req struct {
				JSONRPC string `json:"jsonrpc"`
				ID      uint64 `json:"id"`
				Method  string `json:"method"`
			}
			_ = json.Unmarshal(msg, &req)
			if req.Method != "auth.login_with_api_key" {
				continue
			}
			n := attempts.Add(1)
			var resp []byte
			if n <= failFirst {
				resp = []byte(`{"jsonrpc":"2.0","id":` +
					fmtID(req.ID) +
					`,"error":{"code":-32001,"message":"Method call error",` +
					`"data":{"errname":"EBUSY","reason":"Rate Limit Exceeded"}}}`)
			} else {
				resp = []byte(`{"jsonrpc":"2.0","id":` + fmtID(req.ID) + `,"result":true}`)
			}
			_ = conn.Write(r.Context(), websocket.MessageText, resp)
		}
	}))
	defer srv.Close()

	// Convert http://host:port → ws://host:port for wsclient.New.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	c := &Client{
		baseURL:        wsURL,
		apiKey:         "k",
		pending:        make(map[uint64]chan *rpcResponse),
		closed:         make(chan struct{}),
		dialTimeout:    5 * time.Second,
		requestTimeout: 30 * time.Second,
		RetryPolicy:    DefaultRetryPolicy(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())
	defer c.lifetimeCancel()

	// Dial; the test server's handler will field the auth call.
	if err := c.dial(ctx, wsURL+"/api/current"); err != nil {
		t.Fatalf("dial: %v", err)
	}

	if err := c.authenticate(ctx); err != nil {
		t.Fatalf("authenticate: %v (attempts=%d)", err, attempts.Load())
	}
	if got := attempts.Load(); got != failFirst+1 {
		t.Errorf("attempts = %d, want %d (failFirst=%d + final success)",
			got, failFirst+1, failFirst)
	}
}

// fmtID is a tiny helper to avoid pulling fmt + strconv into the
// inline server response builder.
func fmtID(id uint64) string {
	if id == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for id > 0 {
		i--
		b[i] = byte('0' + id%10)
		id /= 10
	}
	return string(b[i:])
}

// dialIfMissing isn't used here directly, it's the test's
// documentation that we lean on Client's own dial path because
// wsclient.New(ctx, ...) calls both dial AND authenticate; we want
// to drive authenticate in isolation, so we wire up a Client by
// hand and dial it ourselves. Suppress unused-import-warning
// avoidance for the sync import in case the embedded test pattern
// changes.
var _ = sync.Once{}

// TestAuthenticate_CtxCancelDuringBackoff covers the ctx-cancellation
// branch inside the retry backoff select.
func TestAuthenticate_CtxCancelDuringBackoff(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			_, msg, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			_ = json.Unmarshal(msg, &req)
			if req.Method != "auth.login_with_api_key" {
				continue
			}
			// Always rate-limit so the client enters backoff.
			resp := []byte(`{"jsonrpc":"2.0","id":` + fmtID(req.ID) +
				`,"error":{"code":-32001,"message":"Method call error",` +
				`"data":{"errname":"EBUSY","reason":"Rate Limit Exceeded"}}}`)
			_ = conn.Write(r.Context(), websocket.MessageText, resp)
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	c := &Client{
		baseURL:        wsURL,
		apiKey:         "k",
		pending:        make(map[uint64]chan *rpcResponse),
		closed:         make(chan struct{}),
		dialTimeout:    5 * time.Second,
		requestTimeout: 10 * time.Second,
		RetryPolicy:    DefaultRetryPolicy(),
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())
	defer c.lifetimeCancel()

	ctx, cancel := context.WithCancel(context.Background())
	if err := c.dial(ctx, wsURL+"/api/current"); err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Cancel shortly after the first EBUSY lands, the client will be
	// sitting in its jittered backoff select.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := c.authenticate(ctx)
	if err == nil {
		t.Fatal("expected ctx-cancellation error")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context-canceled wrap, got: %v", err)
	}
}

// TestAuthenticate_DelayCapApplies rate-limits enough consecutive
// attempts that the exponential backoff hits the 6s cap branch.
// Slow (~20s of real backoff), kept in the suite because the cap
// branch otherwise never executes.
func TestAuthenticate_DelayCapApplies(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	const failFirst = 7 // delay doubles: .2 .4 .8 1.6 3.2 6.4→cap 6
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			_, msg, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			_ = json.Unmarshal(msg, &req)
			if req.Method != "auth.login_with_api_key" {
				continue
			}
			n := attempts.Add(1)
			var resp []byte
			if n <= failFirst {
				resp = []byte(`{"jsonrpc":"2.0","id":` + fmtID(req.ID) +
					`,"error":{"code":-32001,"message":"Method call error",` +
					`"data":{"errname":"EBUSY","reason":"Rate Limit Exceeded"}}}`)
			} else {
				resp = []byte(`{"jsonrpc":"2.0","id":` + fmtID(req.ID) + `,"result":true}`)
			}
			_ = conn.Write(r.Context(), websocket.MessageText, resp)
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	c := &Client{
		baseURL:        wsURL,
		apiKey:         "k",
		pending:        make(map[uint64]chan *rpcResponse),
		closed:         make(chan struct{}),
		dialTimeout:    5 * time.Second,
		requestTimeout: 30 * time.Second,
		RetryPolicy:    DefaultRetryPolicy(),
	}
	c.lifetime, c.lifetimeCancel = context.WithCancel(context.Background())
	defer c.lifetimeCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := c.dial(ctx, wsURL+"/api/current"); err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := c.authenticate(ctx); err != nil {
		t.Fatalf("authenticate: %v (attempts=%d)", err, attempts.Load())
	}
	if got := attempts.Load(); got != failFirst+1 {
		t.Errorf("attempts = %d, want %d", got, failFirst+1)
	}
}
