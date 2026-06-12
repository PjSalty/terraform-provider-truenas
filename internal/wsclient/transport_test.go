package wsclient

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBackoffDelay(t *testing.T) {
	cases := []struct {
		name    string
		policy  RetryPolicy
		attempt int
		want    time.Duration
	}{
		{"defaults attempt 0", RetryPolicy{}, 0, 500 * time.Millisecond},
		{"defaults attempt 1", RetryPolicy{}, 1, time.Second},
		{"defaults attempt 2", RetryPolicy{}, 2, 2 * time.Second},
		{"defaults capped", RetryPolicy{}, 100, 10 * time.Second},
		{"negative attempt", RetryPolicy{}, -5, 500 * time.Millisecond},
		{"custom base", RetryPolicy{BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}, 2, 400 * time.Millisecond},
		{"shift overflow guard", RetryPolicy{BaseDelay: time.Hour, MaxDelay: 24 * time.Hour}, 22, 24 * time.Hour},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := backoffDelay(tt.policy, tt.attempt)
			if got != tt.want {
				t.Errorf("backoffDelay(%+v, %d) = %v, want %v", tt.policy, tt.attempt, got, tt.want)
			}
		})
	}
}

func TestSleepCtx(t *testing.T) {
	t.Run("returns nil after sleeping", func(t *testing.T) {
		start := time.Now()
		err := sleepCtx(context.Background(), 50*time.Millisecond)
		elapsed := time.Since(start)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if elapsed < 40*time.Millisecond {
			t.Errorf("returned too early: %v", elapsed)
		}
	})

	t.Run("zero duration returns ctx.Err", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := sleepCtx(ctx, 0)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("negative duration with live ctx", func(t *testing.T) {
		err := sleepCtx(context.Background(), -1*time.Second)
		if err != nil {
			t.Errorf("expected nil for negative duration with live ctx, got %v", err)
		}
	})

	t.Run("ctx canceled mid-sleep", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		err := sleepCtx(ctx, time.Hour)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestSendFrame_nilConn(t *testing.T) {
	c := &Client{
		closed:  make(chan struct{}),
		pending: make(map[uint64]chan *rpcResponse),
	}
	err := c.sendFrame(context.Background(), rpcRequest{JSONRPC: "2.0", ID: 1, Method: "x"})
	if !errors.Is(err, ErrConnectionLost) {
		t.Errorf("expected ErrConnectionLost, got %v", err)
	}
}

func TestWrapWriteErr(t *testing.T) {
	cases := []struct {
		name     string
		in       error
		wantBare error // non-nil if errors.Is(out, wantBare) must be true
		wantWrap bool  // true if out should wrap ErrConnectionLost
		wantNil  bool
	}{
		{"nil", nil, nil, false, true},
		{"context.Canceled", context.Canceled, context.Canceled, false, false},
		{"context.DeadlineExceeded", context.DeadlineExceeded, context.DeadlineExceeded, false, false},
		{"random error", errors.New("broken pipe"), nil, true, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out := wrapWriteErr(tt.in)
			if tt.wantNil {
				if out != nil {
					t.Errorf("expected nil, got %v", out)
				}
				return
			}
			if tt.wantBare != nil && !errors.Is(out, tt.wantBare) {
				t.Errorf("errors.Is(out, %v) = false; out = %v", tt.wantBare, out)
			}
			if tt.wantWrap && !errors.Is(out, ErrConnectionLost) {
				t.Errorf("expected ErrConnectionLost wrap, got %v", out)
			}
		})
	}
}

func TestIsClosing(t *testing.T) {
	c := &Client{closed: make(chan struct{})}
	if c.isClosing() {
		t.Error("isClosing on fresh client should be false")
	}
	close(c.closed)
	if !c.isClosing() {
		t.Error("isClosing after close should be true")
	}
}
