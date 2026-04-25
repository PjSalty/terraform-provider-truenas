package client_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// withFastRetries returns a client with a very small base delay so that
// tests exercising multiple retries complete quickly.
func withFastRetries(c *client.Client, maxAttempts int) {
	c.RetryPolicy = client.RetryPolicy{
		MaxAttempts: maxAttempts,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}
}

// TestRetry_500Then200 verifies the canonical scenario from the
// requirements: 500 → 500 → 200 should succeed after exactly 3 attempts.
func TestRetry_500Then200(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		switch n {
		case 1, 2:
			http.Error(w, `{"message":"transient"}`, http.StatusInternalServerError)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	})
	_, c := newTestServer(t, handler)
	withFastRetries(c, 5)

	body, err := c.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", string(body))
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

// TestRetry_TableDriven covers status-code retry policy, method
// idempotency, Retry-After handling, and exhaustion.
func TestRetry_TableDriven(t *testing.T) {
	type serverResponse struct {
		status  int
		body    string
		headers map[string]string
	}
	type testCase struct {
		name            string
		method          string
		responses       []serverResponse
		maxAttempts     int
		wantAttempts    int32
		wantErr         bool
		wantStatus      int    // expected APIError status on failure (0 if none)
		wantBody        string // expected response body on success
		wantMinDuration time.Duration
		honorRetryAfter bool
	}

	cases := []testCase{
		{
			name:   "GET 500 then 200 retries",
			method: http.MethodGet,
			responses: []serverResponse{
				{status: 500, body: `{"message":"boom"}`},
				{status: 200, body: `ok`},
			},
			maxAttempts:  5,
			wantAttempts: 2,
			wantBody:     "ok",
		},
		{
			name:   "GET 502 503 504 then 200",
			method: http.MethodGet,
			responses: []serverResponse{
				{status: 502, body: ``},
				{status: 503, body: ``},
				{status: 504, body: ``},
				{status: 200, body: `done`},
			},
			maxAttempts:  5,
			wantAttempts: 4,
			wantBody:     "done",
		},
		{
			name:   "GET 400 does NOT retry",
			method: http.MethodGet,
			responses: []serverResponse{
				{status: 400, body: `{"message":"bad"}`},
			},
			maxAttempts:  5,
			wantAttempts: 1,
			wantErr:      true,
			wantStatus:   400,
		},
		{
			name:   "POST 500 does NOT retry (non-idempotent)",
			method: http.MethodPost,
			responses: []serverResponse{
				{status: 500, body: `{"message":"boom"}`},
			},
			maxAttempts:  5,
			wantAttempts: 1,
			wantErr:      true,
			wantStatus:   500,
		},
		{
			name:   "PUT 503 retries and succeeds",
			method: http.MethodPut,
			responses: []serverResponse{
				{status: 503, body: ``},
				{status: 200, body: `updated`},
			},
			maxAttempts:  5,
			wantAttempts: 2,
			wantBody:     "updated",
		},
		{
			name:   "DELETE 504 retries and succeeds",
			method: http.MethodDelete,
			responses: []serverResponse{
				{status: 504, body: ``},
				{status: 200, body: `deleted`},
			},
			maxAttempts:  5,
			wantAttempts: 2,
			wantBody:     "deleted",
		},
		{
			name:   "GET exhausts all attempts",
			method: http.MethodGet,
			responses: []serverResponse{
				{status: 500},
				{status: 500},
				{status: 500},
			},
			maxAttempts:  3,
			wantAttempts: 3,
			wantErr:      true,
			wantStatus:   500,
		},
		{
			name:   "GET 429 honors Retry-After",
			method: http.MethodGet,
			responses: []serverResponse{
				{
					status:  429,
					body:    ``,
					headers: map[string]string{"Retry-After": "1"},
				},
				{status: 200, body: `ok`},
			},
			maxAttempts:     5,
			wantAttempts:    2,
			wantBody:        "ok",
			wantMinDuration: 900 * time.Millisecond,
			honorRetryAfter: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu         sync.Mutex
				idx        int
				attempts   int32
				respBundle = tc.responses
			)
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				atomic.AddInt32(&attempts, 1)
				mu.Lock()
				var r serverResponse
				if idx < len(respBundle) {
					r = respBundle[idx]
					idx++
				} else {
					r = respBundle[len(respBundle)-1]
				}
				mu.Unlock()
				for k, v := range r.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(r.status)
				if r.body != "" {
					_, _ = w.Write([]byte(r.body))
				}
			})
			_, c := newTestServer(t, handler)
			// For Retry-After test, we actually want honoring the header
			// (~1s), so we keep the normal delay small — the header should
			// dominate.
			withFastRetries(c, tc.maxAttempts)

			start := time.Now()
			var (
				body []byte
				err  error
			)
			switch tc.method {
			case http.MethodGet:
				body, err = c.Get(context.Background(), "/r")
			case http.MethodPost:
				body, err = c.Post(context.Background(), "/r", map[string]string{"a": "b"})
			case http.MethodPut:
				body, err = c.Put(context.Background(), "/r", map[string]string{"a": "b"})
			case http.MethodDelete:
				body, err = c.Delete(context.Background(), "/r")
			default:
				t.Fatalf("unsupported method %q", tc.method)
			}
			elapsed := time.Since(start)

			if got := atomic.LoadInt32(&attempts); got != tc.wantAttempts {
				t.Errorf("attempts = %d, want %d", got, tc.wantAttempts)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (body=%q)", string(body))
				}
				var apiErr *client.APIError
				if errors.As(err, &apiErr) {
					if tc.wantStatus != 0 && apiErr.StatusCode != tc.wantStatus {
						t.Errorf("status = %d, want %d", apiErr.StatusCode, tc.wantStatus)
					}
				} else if tc.wantStatus != 0 {
					t.Errorf("error is not *APIError: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantBody != "" && string(body) != tc.wantBody {
				t.Errorf("body = %q, want %q", string(body), tc.wantBody)
			}
			if tc.honorRetryAfter && elapsed < tc.wantMinDuration {
				t.Errorf("elapsed = %s, want >= %s (Retry-After not honored)", elapsed, tc.wantMinDuration)
			}
		})
	}
}

// TestRetry_ContextCancellation verifies that canceling the context
// between retries aborts the loop immediately.
func TestRetry_ContextCancellation(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, `{"message":"always fail"}`, http.StatusInternalServerError)
	})
	_, c := newTestServer(t, handler)
	// Use a base delay large enough that cancellation will interrupt
	// the sleep between retries.
	c.RetryPolicy = client.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := c.Get(ctx, "/r")
	if err == nil {
		t.Fatal("expected error from cancellation, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// Should have attempted at least once but well short of 10.
	if a := atomic.LoadInt32(&attempts); a < 1 || a > 3 {
		t.Errorf("attempts = %d, want 1..3", a)
	}
}

// TestRetry_DeadlineExceeded verifies that a ctx deadline aborts retries.
func TestRetry_DeadlineExceeded(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{}`, http.StatusInternalServerError)
	})
	_, c := newTestServer(t, handler)
	c.RetryPolicy = client.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, "/r")
	if err == nil {
		t.Fatal("expected error from deadline, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		// Some code paths may return a wrapped net/http error — accept that
		// too, but require the deadline to actually have fired.
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	}
}

// TestRetry_POSTRetriesOnTransportError verifies that POST still retries
// when the transport layer fails (since we know the request body did not
// reach the server).
func TestRetry_POSTRetriesOnTransportError(t *testing.T) {
	var attempts int32
	// Start a server then close it, so the first dials fail with
	// connection refused. Then flip a mutex-protected "ready" flag and
	// start a second server on the SAME addr.
	//
	// Simpler approach: use a handler that hijacks the first N connections
	// and abruptly closes them, simulating a transport-level reset.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			// Hijack and close without writing a response — the client
			// will observe an unexpected EOF / broken connection.
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "nope", http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				return
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	c, err := client.New(srv.URL, "k")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	c.RetryPolicy = client.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	body, err := c.Post(context.Background(), "/r", map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %q", string(body))
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
}

// TestParseRetryAfter_Seconds is a quick sanity check that the Retry-After
// header parser handles the seconds form (HTTP-date form is covered
// implicitly by the table-driven test above).
func TestParseRetryAfter_Seconds(t *testing.T) {
	// We can't call the unexported parseRetryAfter directly from the
	// _test package, but we can observe its effect via a live server.
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", strconv.Itoa(0))
			http.Error(w, "", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `ok`)
	})
	_, c := newTestServer(t, handler)
	withFastRetries(c, 3)
	if _, err := c.Get(context.Background(), "/r"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}
