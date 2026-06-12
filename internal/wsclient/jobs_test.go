package wsclient

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCallJob_success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pollCount atomic.Int64
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "pool.dataset.lock":
			return float64(123), nil // job ID
		case "core.get_jobs":
			n := pollCount.Add(1)
			state := JobStateRunning
			if n >= 2 {
				state = JobStateSuccess
			}
			return []map[string]interface{}{{
				"id":     123,
				"state":  state,
				"result": "locked",
			}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := c.CallJob(ctx, "pool.dataset.lock", []interface{}{"tank/foo"}, CallOptions{Job: true}, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("CallJob: %v", err)
	}
	var s string
	if err := json.Unmarshal(result, &s); err != nil {
		t.Fatalf("Unmarshal result: %v", err)
	}
	if s != "locked" {
		t.Errorf("result: got %q, want %q", s, "locked")
	}
	if pollCount.Load() < 2 {
		t.Errorf("expected at least 2 poll calls, got %d", pollCount.Load())
	}
}

func TestCallJob_failed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "pool.scrub.start":
			return float64(7), nil
		case "core.get_jobs":
			return []map[string]interface{}{{
				"id":    7,
				"state": JobStateFailed,
				"error": "scrub aborted: pool offline",
			}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CallJob(ctx, "pool.scrub.start", nil, CallOptions{}, 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "scrub aborted") {
		t.Errorf("expected job-failed error, got %v", err)
	}
}

func TestCallJob_aborted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "x.run":
			return float64(1), nil
		case "core.get_jobs":
			return []map[string]interface{}{{"id": 1, "state": JobStateAborted}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CallJob(ctx, "x.run", nil, CallOptions{}, 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Errorf("expected aborted error, got %v", err)
	}
}

func TestCallJob_unknownState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "x.run":
			return float64(1), nil
		case "core.get_jobs":
			return []map[string]interface{}{{"id": 1, "state": "WEIRD"}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CallJob(ctx, "x.run", nil, CallOptions{}, 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "unknown job state") {
		t.Errorf("expected unknown-state error, got %v", err)
	}
}

func TestCallJob_initialMethodFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "bad"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CallJob(ctx, "x.run", nil, CallOptions{}, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected RPC error")
	}
}

func TestCallJob_nonIntegerJobID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		// Server returns a string instead of a numeric job ID.
		return "not-a-job-id", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CallJob(ctx, "x.run", nil, CallOptions{}, 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "did not return a job ID") {
		t.Errorf("expected job-ID decode error, got %v", err)
	}
}

func TestCallJob_zeroPollDefaults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "x.run":
			return float64(1), nil
		case "core.get_jobs":
			return []map[string]interface{}{{"id": 1, "state": JobStateSuccess, "result": "ok"}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// pollInterval=0 must fall back to the default 2s; we overcome it
	// by short-circuiting the SUCCESS state on first poll.
	_, err = c.CallJob(ctx, "x.run", nil, CallOptions{}, 0)
	if err != nil {
		t.Fatalf("CallJob: %v", err)
	}
}

func TestCallJob_ctxCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "x.run":
			return float64(1), nil
		case "core.get_jobs":
			return []map[string]interface{}{{"id": 1, "state": JobStateRunning}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// pollInterval > cancel delay so cancel definitely lands while
	// CallJob is parked in sleepCtx, hitting the sleepCtx-err branch.
	callCtx, callCancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		callCancel()
	}()
	_, err = c.CallJob(callCtx, "x.run", nil, CallOptions{}, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected ctx canceled")
	}
}

func TestGetJob_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetJob(ctx, 99)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestGetJob_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not an array", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetJob(ctx, 99)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected decode error, got %v", err)
	}
}
