package wsclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestNew_validation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		baseURL string
		apiKey  string
		wantErr string
	}{
		{"empty base URL", "", "k", "base URL is required"},
		{"empty key", "https://x", "", "API key is required"},
		{"bad scheme", "ftp://x", "k", "unsupported URL scheme"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(ctx, tt.baseURL, tt.apiKey, false)
			if c != nil {
				_ = c.Close()
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestNew_dialFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// 127.0.0.1:1 is reserved, dial will fail fast.
	c, err := New(ctx, "http://127.0.0.1:1", "key", true)
	if c != nil {
		_ = c.Close()
	}
	if err == nil {
		t.Fatal("expected dial failure, got nil")
	}
}

func TestSetRequestTimeout(t *testing.T) {
	c := &Client{requestTimeout: DefaultRequestTimeout}
	c.SetRequestTimeout(2 * time.Second)
	if c.requestTimeout != 2*time.Second {
		t.Errorf("SetRequestTimeout(2s): got %v", c.requestTimeout)
	}
	c.SetRequestTimeout(0)
	if c.requestTimeout != DefaultRequestTimeout {
		t.Errorf("SetRequestTimeout(0): expected default %v, got %v", DefaultRequestTimeout, c.requestTimeout)
	}
	c.SetRequestTimeout(-5 * time.Second)
	if c.requestTimeout != DefaultRequestTimeout {
		t.Errorf("SetRequestTimeout(neg): expected default, got %v", c.requestTimeout)
	}
}

func TestSetRetryPolicy(t *testing.T) {
	c := &Client{retryPolicy: DefaultRetryPolicy()}
	c.SetRetryPolicy(RetryPolicy{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: 5 * time.Second})
	if c.retryPolicy.MaxAttempts != 3 || c.retryPolicy.BaseDelay != time.Second || c.retryPolicy.MaxDelay != 5*time.Second {
		t.Errorf("SetRetryPolicy: got %+v", c.retryPolicy)
	}
	// Zero falls back to defaults.
	c.SetRetryPolicy(RetryPolicy{})
	if c.retryPolicy != DefaultRetryPolicy() {
		t.Errorf("zero policy: expected defaults, got %+v", c.retryPolicy)
	}
}

func TestClose_idempotent(t *testing.T) {
	c := &Client{closed: make(chan struct{}), pending: map[uint64]chan *rpcResponse{}}
	if err := c.Close(); err != nil {
		t.Errorf("Close 1: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close 2 (idempotent): %v", err)
	}
}

func TestFailPending(t *testing.T) {
	c := &Client{pending: make(map[uint64]chan *rpcResponse)}
	ch1 := make(chan *rpcResponse, 1)
	ch2 := make(chan *rpcResponse, 1)
	c.pending[1] = ch1
	c.pending[2] = ch2

	sentinel := errors.New("test failure")
	c.failPending(sentinel)

	if len(c.pending) != 0 {
		t.Errorf("pending should be empty, got %d", len(c.pending))
	}
	// failPending now passes the error through transportErr (preserving
	// the error chain so callers can errors.Is against ErrConnectionLost
	// / ErrShuttingDown). The Error field stays nil — wire-level
	// RPCErrors are a different code path.
	r1, ok1 := <-ch1
	if !ok1 || r1 == nil || r1.transportErr == nil || !errors.Is(r1.transportErr, sentinel) {
		t.Errorf("ch1: got %+v ok=%v", r1, ok1)
	}
	if r1 != nil && r1.Error != nil {
		t.Errorf("ch1: Error should be nil for synthetic transport failures, got %v", r1.Error)
	}
	r2, ok2 := <-ch2
	if !ok2 || r2 == nil || r2.transportErr == nil {
		t.Errorf("ch2: got %+v ok=%v", r2, ok2)
	}
}

func TestFailPending_fullChannel(t *testing.T) {
	c := &Client{pending: make(map[uint64]chan *rpcResponse)}
	// Channel with cap=1, pre-filled — failPending's send must drop, not block.
	ch := make(chan *rpcResponse, 1)
	ch <- &rpcResponse{ID: 99}
	c.pending[1] = ch

	done := make(chan struct{})
	go func() {
		c.failPending(errors.New("fail"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("failPending blocked on full channel")
	}
}

func TestNextRequestID_monotonic(t *testing.T) {
	c := &Client{}
	a := c.nextRequestID()
	b := c.nextRequestID()
	if b <= a {
		t.Errorf("expected monotonic; got a=%d b=%d", a, b)
	}
}

func TestNewCorrelationID(t *testing.T) {
	id := newCorrelationID()
	if len(id) != 16 {
		t.Errorf("expected 16-char ID, got %q (%d)", id, len(id))
	}
	if id == newCorrelationID() {
		t.Error("expected unique IDs across calls")
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestNewCorrelationIDFrom_failure(t *testing.T) {
	id := newCorrelationIDFrom(errReader{})
	if id == "" {
		t.Error("fallback ID should be non-empty")
	}
	// Fallback uses time.Now().UnixNano() as hex; should be parseable.
	if len(id) < 8 {
		t.Errorf("fallback too short: %q", id)
	}
}

func TestNewCorrelationIDFrom_partial(t *testing.T) {
	// Read returns short — io.ReadFull surfaces ErrUnexpectedEOF.
	r := bytes.NewReader([]byte{1, 2, 3})
	id := newCorrelationIDFrom(r)
	if id == "" {
		t.Error("expected fallback non-empty")
	}
}

func TestWSURLFromBase(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"https://truenas.example.com", "wss://truenas.example.com/api/current", false},
		{"http://10.10.40.10", "ws://10.10.40.10/api/current", false},
		{"https://truenas.example.com/", "wss://truenas.example.com/api/current", false},
		{"https://truenas.example.com/api/v2.0", "wss://truenas.example.com/api/current", false},
		{"https://truenas.example.com/api/v2.0/", "wss://truenas.example.com/api/current", false},
		{"wss://truenas.example.com", "wss://truenas.example.com/api/current", false},
		{"ws://truenas.example.com/api/v2.0", "ws://truenas.example.com/api/current", false},
		{"", "", true},
		{"ftp://x", "", true},
		{"telnet://10.10.40.10", "", true},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			got, err := wsURLFromBase(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.in)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("wsURLFromBase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
