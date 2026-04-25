package client

// Direct (same-package) tests for unexported helpers whose error branches
// are otherwise hard to reach through the public API.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --------------------------------------------------------------------
// isRetryableTransportErr — every branch.
// --------------------------------------------------------------------

type fakeNetTimeout struct{ msg string }

func (f fakeNetTimeout) Error() string   { return f.msg }
func (f fakeNetTimeout) Timeout() bool   { return true }
func (f fakeNetTimeout) Temporary() bool { return true }

// Ensure fakeNetTimeout satisfies net.Error.
var _ net.Error = fakeNetTimeout{}

func TestIsRetryableTransportErr_AllBranches(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"io.EOF", io.EOF, true},
		{"io.ErrUnexpectedEOF", io.ErrUnexpectedEOF, true},
		{"net.Error timeout", fakeNetTimeout{msg: "dial tcp: i/o timeout"}, true},
		{"connection reset string match", errors.New("read tcp 127.0.0.1: connection reset by peer"), true},
		{"connection refused string match", errors.New("dial tcp 127.0.0.1: connection refused"), true},
		{"broken pipe string match", errors.New("write: broken pipe"), true},
		{"EOF string match (wrapped)", fmt.Errorf("read: %w", errors.New("some EOF happened")), true},
		{"unrelated error", errors.New("this is an unrelated error"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableTransportErr(tc.err); got != tc.want {
				t.Errorf("isRetryableTransportErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// --------------------------------------------------------------------
// parseRetryAfter — nil response, empty header, seconds, HTTP-date,
// past-date, unparseable.
// --------------------------------------------------------------------

func TestParseRetryAfter_AllBranches(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		if d := parseRetryAfter(nil); d != 0 {
			t.Errorf("parseRetryAfter(nil) = %v, want 0", d)
		}
	})
	t.Run("empty header", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		if d := parseRetryAfter(resp); d != 0 {
			t.Errorf("empty = %v, want 0", d)
		}
	})
	t.Run("seconds form", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "5")
		if d := parseRetryAfter(resp); d != 5*time.Second {
			t.Errorf("seconds = %v, want 5s", d)
		}
	})
	t.Run("HTTP-date form future", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
		resp.Header.Set("Retry-After", future)
		if d := parseRetryAfter(resp); d <= 0 {
			t.Errorf("future date = %v, want > 0", d)
		}
	})
	t.Run("HTTP-date form past", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		past := time.Now().Add(-30 * time.Second).UTC().Format(http.TimeFormat)
		resp.Header.Set("Retry-After", past)
		// Past dates return 0 (the delay has already elapsed).
		if d := parseRetryAfter(resp); d != 0 {
			t.Errorf("past date = %v, want 0", d)
		}
	})
	t.Run("unparseable", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "not-a-number-or-date")
		if d := parseRetryAfter(resp); d != 0 {
			t.Errorf("unparseable = %v, want 0", d)
		}
	})
	t.Run("negative seconds", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "-5")
		// Negative seconds are rejected (secs >= 0 check).
		if d := parseRetryAfter(resp); d != 0 {
			t.Errorf("negative = %v, want 0", d)
		}
	})
}

// --------------------------------------------------------------------
// backoffDelay — zero defaults, overflow, normal progression.
// --------------------------------------------------------------------

func TestBackoffDelay_AllBranches(t *testing.T) {
	t.Run("zero base/max defaults", func(t *testing.T) {
		d := backoffDelay(RetryPolicy{}, 0)
		// Must return something > 0 once defaults kick in.
		if d <= 0 {
			t.Errorf("d = %v, want > 0", d)
		}
	})
	t.Run("shift cap at 20", func(t *testing.T) {
		// attempt=100 exceeds shift cap (20) and hits the max-delay cap.
		d := backoffDelay(RetryPolicy{BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}, 100)
		if d > 15*time.Millisecond {
			t.Errorf("d = %v, want <= ~15ms (capped)", d)
		}
	})
	t.Run("normal progression", func(t *testing.T) {
		p := RetryPolicy{BaseDelay: 10 * time.Millisecond, MaxDelay: 1 * time.Second}
		d0 := backoffDelay(p, 0)
		d2 := backoffDelay(p, 2)
		if d0 <= 0 || d2 <= 0 {
			t.Errorf("d0=%v d2=%v", d0, d2)
		}
	})
	t.Run("negative attempt clamps to zero", func(t *testing.T) {
		p := RetryPolicy{BaseDelay: 10 * time.Millisecond, MaxDelay: 1 * time.Second}
		d := backoffDelay(p, -5)
		if d != 10*time.Millisecond {
			t.Errorf("d = %v, want 10ms (base delay at attempt=0)", d)
		}
	})
}

// --------------------------------------------------------------------
// sleepCtx — zero/negative, normal completion, ctx cancel.
// --------------------------------------------------------------------

func TestSleepCtx_AllBranches(t *testing.T) {
	t.Run("zero duration returns ctx.Err()", func(t *testing.T) {
		// No deadline, no cancel -> ctx.Err()==nil -> returns nil.
		if err := sleepCtx(context.Background(), 0); err != nil {
			t.Errorf("want nil, got %v", err)
		}
	})
	t.Run("negative duration with canceled ctx returns err", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := sleepCtx(ctx, -1*time.Second)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("want Canceled, got %v", err)
		}
	})
	t.Run("normal completion", func(t *testing.T) {
		start := time.Now()
		if err := sleepCtx(context.Background(), 5*time.Millisecond); err != nil {
			t.Errorf("want nil, got %v", err)
		}
		if time.Since(start) < 4*time.Millisecond {
			t.Errorf("slept less than expected")
		}
	})
	t.Run("ctx canceled during sleep", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()
		err := sleepCtx(ctx, 1*time.Second)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("want Canceled, got %v", err)
		}
	})
}

// --------------------------------------------------------------------
// NewWithOptions insecureSkipVerify=true path.
// --------------------------------------------------------------------

func TestNewWithOptions_InsecureSkipVerify(t *testing.T) {
	c, err := NewWithOptions("http://truenas.example.com", "key", true)
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	// We don't inspect internal httpClient.Transport — creating the client
	// with insecureSkipVerify=true is enough to exercise the branch.
}

// --------------------------------------------------------------------
// WaitForJob error branches: GET error, invalid JSON, empty jobs array,
// ctx cancel while running, polling loop still-running to success.
// --------------------------------------------------------------------

func TestWaitForJob_AllBranches(t *testing.T) {
	t.Run("GET error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
		}))
		defer srv.Close()
		c, err := New(srv.URL, "k")
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		c.RetryPolicy = RetryPolicy{MaxAttempts: 1, BaseDelay: 1 * time.Millisecond, MaxDelay: 2 * time.Millisecond}
		if _, err := c.WaitForJob(context.Background(), 1); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()
		c, _ := New(srv.URL, "k")
		if _, err := c.WaitForJob(context.Background(), 1); err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("empty jobs array", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
		}))
		defer srv.Close()
		c, _ := New(srv.URL, "k")
		if _, err := c.WaitForJob(context.Background(), 99); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("context canceled while polling", func(t *testing.T) {
		// Job never reaches a terminal state — always returns "RUNNING".
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"state":"RUNNING"}]`))
		}))
		defer srv.Close()
		c, _ := New(srv.URL, "k")
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		_, err := c.WaitForJob(ctx, 1)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("want Canceled, got %v", err)
		}
	})
}

// --------------------------------------------------------------------
// doRequest: request-creation error and body-marshal error branches.
// --------------------------------------------------------------------

type badJSON struct{}

func (badJSON) MarshalJSON() ([]byte, error) { return nil, errors.New("marshal boom") }

func TestDoRequest_MarshalError(t *testing.T) {
	c, _ := New("http://example.com", "k")
	_, err := c.doRequest(context.Background(), http.MethodPost, "/x", badJSON{})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestDoRequest_RequestCreationError(t *testing.T) {
	c, _ := New("http://example.com", "k")
	// An invalid HTTP method with control chars fails http.NewRequestWithContext.
	_, err := c.doRequest(context.Background(), "GE T\n", "/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDoRequest_CtxCanceledBeforeFirstAttempt(t *testing.T) {
	c, _ := New("http://example.com", "k")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.doRequest(ctx, http.MethodGet, "/x", nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want Canceled, got %v", err)
	}
}

func TestDoRequest_MaxAttemptsZeroClamped(t *testing.T) {
	// MaxAttempts < 1 should be treated as 1 — a single attempt.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	c, _ := New(srv.URL, "k")
	c.RetryPolicy = RetryPolicy{MaxAttempts: 0}
	body, err := c.doRequest(context.Background(), http.MethodGet, "/x", nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q", body)
	}
}

// doRequest transport-error path: we start a server, close it, then use
// its URL so Dial fails with "connection refused" — which isRetryableTransportErr
// reports as retryable. POST should retry at least once then give up.
func TestDoRequest_POSTTransportErrorExhausts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close() // Now dials to url will fail.
	c, _ := New(url, "k")
	c.RetryPolicy = RetryPolicy{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond, MaxDelay: 2 * time.Millisecond}
	_, err := c.Post(context.Background(), "/x", map[string]string{})
	if err == nil {
		t.Fatal("expected transport error")
	}
}

// doRequest context canceled during transport-error backoff.
func TestDoRequest_CtxCanceledDuringBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close() // Dials now fail.
	c, _ := New(url, "k")
	c.RetryPolicy = RetryPolicy{MaxAttempts: 5, BaseDelay: 100 * time.Millisecond, MaxDelay: 200 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err := c.Get(ctx, "/x")
	if err == nil {
		t.Fatal("expected error")
	}
}

// brokenBody implements io.ReadCloser and always returns an EOF-style
// error on Read, simulating a transport that terminates mid-body.
type brokenBody struct{}

func (brokenBody) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (brokenBody) Close() error               { return nil }

// brokenBodyTransport wraps a Response so that reading the body fails.
type brokenBodyTransport struct{ attempts *int }

func (b brokenBodyTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	*b.attempts++
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       brokenBody{},
		Header:     http.Header{},
	}
	return resp, nil
}

// TestDoRequest_ReadBodyRetryable covers lines 395-403 where io.ReadAll
// returns a retryable error. We retry until exhaustion and then return
// the last-seen read error.
func TestDoRequest_ReadBodyRetryable(t *testing.T) {
	attempts := 0
	c, _ := New("http://example.com", "k")
	c.httpClient = &http.Client{Transport: brokenBodyTransport{attempts: &attempts}}
	c.RetryPolicy = RetryPolicy{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 2 * time.Millisecond}

	_, err := c.doRequest(context.Background(), http.MethodGet, "/x", nil)
	if err == nil {
		t.Fatal("expected read error")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

// nonRetryableReadBody is a ReadCloser whose Read fails with a non-retryable
// error (not net.Error, not EOF, no string match).
type nonRetryableReadBody struct{}

func (nonRetryableReadBody) Read(_ []byte) (int, error) {
	return 0, errors.New("synthetic non-retryable read error")
}
func (nonRetryableReadBody) Close() error { return nil }

type nonRetryableBodyTransport struct{ attempts *int }

func (n nonRetryableBodyTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	*n.attempts++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       nonRetryableReadBody{},
		Header:     http.Header{},
	}, nil
}

// TestDoRequest_NonRetryableReadError covers the early-exit branch when
// the response body read error is not retryable (client.go:400-402).
func TestDoRequest_NonRetryableReadError(t *testing.T) {
	attempts := 0
	c, _ := New("http://example.com", "k")
	c.httpClient = &http.Client{Transport: nonRetryableBodyTransport{attempts: &attempts}}
	c.RetryPolicy = RetryPolicy{MaxAttempts: 5}
	_, err := c.doRequest(context.Background(), http.MethodGet, "/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on non-retryable)", attempts)
	}
}

// TestDoRequest_ReadBodyRetryCtxCanceled covers the sleepCtx failure path
// inside the readErr retry loop (line 398-400).
func TestDoRequest_ReadBodyRetryCtxCanceled(t *testing.T) {
	attempts := 0
	c, _ := New("http://example.com", "k")
	c.httpClient = &http.Client{Transport: brokenBodyTransport{attempts: &attempts}}
	// Large backoff ensures ctx cancels before the sleep finishes.
	c.RetryPolicy = RetryPolicy{MaxAttempts: 5, BaseDelay: 500 * time.Millisecond, MaxDelay: 1 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err := c.doRequest(ctx, http.MethodGet, "/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --------------------------------------------------------------------
// WaitForJob: RUNNING once then SUCCESS — covers the time.After branch
// at line 206.
// --------------------------------------------------------------------

func TestWaitForJob_RunningThenSuccess(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`[{"id":1,"state":"RUNNING"}]`))
			return
		}
		_, _ = w.Write([]byte(`[{"id":1,"state":"SUCCESS"}]`))
	}))
	defer srv.Close()
	c, _ := New(srv.URL, "k")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	job, err := c.WaitForJob(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.State != "SUCCESS" {
		t.Errorf("state = %q", job.State)
	}
}

// doRequest context canceled mid-Do -> returns ctx.Err() (line 378-380).
func TestDoRequest_CtxCanceledDuringDo(t *testing.T) {
	// Server blocks forever so ctx cancellation interrupts Do().
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	c, _ := New(srv.URL, "k")
	c.RetryPolicy = RetryPolicy{MaxAttempts: 1}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := c.Get(ctx, "/x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want Canceled, got %v", err)
	}
}
