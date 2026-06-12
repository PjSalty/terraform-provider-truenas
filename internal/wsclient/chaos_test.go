package wsclient

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Chaos tests for the WebSocket transport.
//
// These exercise the worst-case scenario the Phase 3 plan calls out:
// the server vanishes mid-call, the client must notice within its
// dial timeout, redial, and resume work. They are unit tests against
// TestServer rather than integration tests against a real TrueNAS,
// because the failure surface we care about is in the client's
// reconnect machinery, not in middlewared.
//
// The Phase 3 plan also calls for a live chaos test (kill -9 on
// middlewared during a real terraform apply); that lives in the
// runbook for examples/prod-smoke and is performed by an operator
// during the v2.0.0-rc.1 → v2.0.0 soak window.

// TestChaos_JobPoll_ServerDropsMidPoll simulates the canonical
// middlewared-restart scenario: a CallJob is in flight, the server
// hangs up after returning the job ID, and the client must reconnect
// to continue polling core.get_jobs.
//
// Outcome: CallJob returns the job's terminal SUCCESS result and the
// follow-up read works. The transient connection drop is invisible
// to the caller.
func TestChaos_JobPoll_ServerDropsMidPoll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const jobID = int64(77)
	pollCount := atomic.Int64{}

	// We need a handle to the TestServer inside its handler so we
	// can call CloseConnection after the first poll. Declare with
	// a forward reference and assign immediately.
	var tsRef *TestServer

	tsRef = NewTestServer(t, func(_ context.Context, method string, _ []interface{}) (interface{}, *RPCError) {
		switch method {
		case "long.run":
			return jobID, nil
		case "core.get_jobs":
			n := pollCount.Add(1)
			if n == 1 {
				// First poll: return RUNNING, then drop the
				// connection on the way out. The next poll
				// must come over a fresh socket.
				go func() {
					// Tiny delay so the response frame has a
					// chance to flush before we tear the conn
					// down. Without this the client gets
					// ErrConnectionLost on the IN-FLIGHT call,
					// not on the NEXT call, which is a
					// different code path we cover separately.
					time.Sleep(50 * time.Millisecond)
					tsRef.CloseConnection()
				}()
				return []map[string]interface{}{{
					"id": jobID, "state": JobStateRunning,
				}}, nil
			}
			// Subsequent polls (post-reconnect) see SUCCESS.
			return []map[string]interface{}{{
				"id":     jobID,
				"state":  JobStateSuccess,
				"result": "done",
			}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})

	c, err := tsRef.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Tighten the poll interval so the test doesn't take 6s sleeping
	// between polls.
	result, err := c.CallJob(ctx, "long.run", nil,
		CallOptions{Job: true, Idempotent: true},
		50*time.Millisecond)
	if err != nil {
		t.Fatalf("CallJob: %v", err)
	}
	if string(result) != `"done"` {
		t.Errorf("result = %s, want \"done\"", string(result))
	}

	// Verify the chaos scenario actually happened: poll count > 1
	// (proving the second poll was issued AFTER the disconnect, not
	// just folded into a single round-trip).
	if pollCount.Load() < 2 {
		t.Errorf("pollCount = %d, want >= 2 (test did not exercise reconnect path)",
			pollCount.Load())
	}
}

// TestChaos_NonIdempotentCall_DropMidFlight covers the other side of
// the reconnect contract: a non-idempotent call (CallOptions{
// Idempotent: false}) that's in flight when the server hangs up must
// surface a clear error rather than silently re-issue the call.
//
// This protects callers from at-least-once semantics on operations
// that don't tolerate double-execution (e.g. *.create paths). The
// caller is expected to retry the *plan* explicitly, which is what
// `terraform apply` does naturally.
func TestChaos_NonIdempotentCall_DropMidFlight(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var tsRef *TestServer
	callCount := atomic.Int64{}

	tsRef = NewTestServer(t, func(_ context.Context, method string, _ []interface{}) (interface{}, *RPCError) {
		// Drop the conn instead of responding. The client should
		// observe ErrConnectionLost and propagate it up because
		// Idempotent: false suppresses the auto-retry.
		callCount.Add(1)
		go func() {
			time.Sleep(20 * time.Millisecond)
			tsRef.CloseConnection()
		}()
		// Block forever; the close above will tear us down.
		<-ctx.Done()
		return nil, nil
	})

	c, err := tsRef.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Call(ctx, "thing.create", nil,
		CallOptions{Idempotent: false, Timeout: 2 * time.Second})
	if err == nil {
		t.Fatal("expected error from non-idempotent call interrupted by drop")
	}
	// Idempotent=false should NOT retry, so the server saw at most
	// one call. (Race-tolerant: the goroutine could spawn just as
	// the test handler exits, but the upper bound is 1.)
	if got := callCount.Load(); got > 1 {
		t.Errorf("non-idempotent call retried: callCount = %d, want 1", got)
	}
}

// TestChaos_IdempotentCall_DropMidFlight is the symmetric positive
// case: an idempotent call (read-only) interrupted by a drop should
// be retried transparently after reconnect.
func TestChaos_IdempotentCall_DropMidFlight(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tsRef *TestServer
	callCount := atomic.Int64{}

	tsRef = NewTestServer(t, func(_ context.Context, method string, _ []interface{}) (interface{}, *RPCError) {
		n := callCount.Add(1)
		if n == 1 {
			// First call: drop the conn before responding. The
			// retry path should kick in.
			go func() {
				time.Sleep(20 * time.Millisecond)
				tsRef.CloseConnection()
			}()
			<-ctx.Done()
			return nil, nil
		}
		// Second call (post-reconnect): respond normally.
		return "ok", nil
	})

	c, err := tsRef.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := c.Call(ctx, "thing.query", nil,
		CallOptions{Read: true, Idempotent: true, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("idempotent call after reconnect: %v", err)
	}
	if !strings.Contains(string(result), "ok") {
		t.Errorf("result = %s, want \"ok\"", string(result))
	}
	if got := callCount.Load(); got != 2 {
		t.Errorf("idempotent call did not retry: callCount = %d, want 2", got)
	}
}
