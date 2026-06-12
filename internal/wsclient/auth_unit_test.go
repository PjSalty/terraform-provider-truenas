package wsclient

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// TestAuthJitterDelay_Range checks that authJitterDelay produces a
// value in [base, base*3) and never exceeds the cap. Repeats enough
// times to make the random-distribution check probabilistically
// solid; the loop is bounded so a flaky-RNG edge can't hang CI.
func TestAuthJitterDelay_Range(t *testing.T) {
	t.Parallel()
	const (
		base = 200 * time.Millisecond
		max  = 6 * time.Second
		runs = 1024
	)
	var minSeen = time.Hour
	var maxSeen time.Duration
	for i := 0; i < runs; i++ {
		d := authJitterDelay(base, max)
		if d < base {
			t.Fatalf("delay %v < base %v (run %d)", d, base, i)
		}
		// authJitterDelay's spec is [base, base*3); the cap kicks in
		// if base*3 > max but for these values base*3 = 600ms which
		// is well under max=6s, so the cap doesn't apply.
		if d >= 3*base {
			t.Fatalf("delay %v >= base*3 %v (run %d)", d, 3*base, i)
		}
		if d > max {
			t.Fatalf("delay %v > max %v (run %d)", d, max, i)
		}
		if d < minSeen {
			minSeen = d
		}
		if d > maxSeen {
			maxSeen = d
		}
	}
	// Sanity: we should have seen a reasonable spread, not the
	// same value 1024 times. Asserts the random source is being
	// consulted.
	if minSeen == maxSeen {
		t.Errorf("authJitterDelay produced the same value %v across %d runs", minSeen, runs)
	}
}

// TestAuthJitterDelay_CapApplies verifies the cap is honoured when
// the jittered upper bound would exceed it.
func TestAuthJitterDelay_CapApplies(t *testing.T) {
	t.Parallel()
	base := 3 * time.Second
	max := 4 * time.Second
	// base*3 = 9s > max; some runs will land above max and get
	// capped. Loop enough to hit one of those.
	for i := 0; i < 256; i++ {
		d := authJitterDelay(base, max)
		if d > max {
			t.Fatalf("delay %v exceeded cap %v (run %d)", d, max, i)
		}
	}
}

// TestAuthJitterDelay_ZeroBase returns zero.
func TestAuthJitterDelay_ZeroBase(t *testing.T) {
	t.Parallel()
	if d := authJitterDelay(0, time.Second); d != 0 {
		t.Errorf("authJitterDelay(0, ...) = %v, want 0", d)
	}
}

// TestIsAuthRateLimited_Matrix walks every classification path.
func TestIsAuthRateLimited_Matrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"non-rpc", errors.New("plain error"), false},
		{"wrong code", &RPCError{Code: CodeMethodNotFound, Message: "anything"}, false},
		{"call-error EBUSY data", &RPCError{
			Code: CodeMethodCallError,
			Data: json.RawMessage(`{"errname":"EBUSY","reason":"Rate Limit Exceeded"}`),
		}, true},
		{"call-error other errname", &RPCError{
			Code: CodeMethodCallError,
			Data: json.RawMessage(`{"errname":"EINVAL","reason":"bad input"}`),
		}, false},
		{"message text fallback rate limit", &RPCError{
			Code:    CodeMethodCallError,
			Message: "service refused: Rate Limit Exceeded",
		}, true},
		{"message text no rate limit phrase", &RPCError{
			Code:    CodeMethodCallError,
			Message: "transient internal error",
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAuthRateLimited(tc.err); got != tc.want {
				t.Errorf("isAuthRateLimited(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestNewWithOptions_ShimReachesNew verifies the compat shim
// delegates to New. Uses an obviously-unreachable host so the dial
// fails fast; we only care that the call shape is accepted.
func TestNewWithOptions_ShimReachesNew(t *testing.T) {
	_, err := NewWithOptions("https://203.0.113.42:1", "k", true)
	if err == nil {
		t.Fatal("NewWithOptions should fail dial against unreachable host")
	}
}

// TestRequestTimeout_Getter verifies the RequestTimeout getter
// returns the configured timeout, including after SetRequestTimeout.
func TestRequestTimeout_Getter(t *testing.T) {
	t.Parallel()
	c := &Client{requestTimeout: DefaultRequestTimeout}
	if got := c.RequestTimeout(); got != DefaultRequestTimeout {
		t.Errorf("got %v, want default %v", got, DefaultRequestTimeout)
	}
	c.SetRequestTimeout(2 * time.Minute)
	if got := c.RequestTimeout(); got != 2*time.Minute {
		t.Errorf("got %v, want 2m", got)
	}
	c.SetRequestTimeout(0) // zero falls back
	if got := c.RequestTimeout(); got != DefaultRequestTimeout {
		t.Errorf("got %v, want default after 0", got)
	}
}

// TestIsNotFound_InvalidParamsENOENT exercises the new branch that
// accepts CodeInvalidParams with a "[ENOENT]" message, the shape
// TrueNAS emits on cronjob.get_instance for missing ids.
func TestIsNotFound_InvalidParamsENOENT(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"InvalidParams [ENOENT] in message", &RPCError{
			Code:    CodeInvalidParams,
			Message: "Invalid params: [ENOENT] None: CronJob 45 does not exist (EINVAL)",
		}, true},
		{"InvalidParams without ENOENT", &RPCError{
			Code:    CodeInvalidParams,
			Message: "Invalid params: schedule_minute must be set",
		}, false},
		{"InvalidParams with [EINVAL] does-not-exist reason", &RPCError{
			Code: CodeInvalidParams,
			Data: json.RawMessage(`{"errname":"EINVAL","reason":"no such object"}`),
		}, true},
		{"unrelated InvalidParams", &RPCError{
			Code:    CodeInvalidParams,
			Message: "Invalid params: count must be positive",
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsNotFound_MatchNotFound covers the v2.0 fix for the surface
// TrueNAS surfaces when *.get_instance can't find the id: the
// "MatchNotFound()" exception class name leaks into the RPC error
// message under CodeMethodCallError without populating Data.
//
// Surfaced live on acc-ws-6 against SCALE 25.10, destroyed Dataset
// + Zvol + Cronjob fixtures all returned this shape, causing
// 9 destroy-check failures.
func TestIsNotFound_MatchNotFound(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"dataset.get_instance after delete (reason field)", &RPCError{
			Code:    CodeMethodCallError,
			Message: "Method call error",
			Data:    json.RawMessage(`{"errname":"EINVAL","reason":"MatchNotFound()"}`),
		}, true},
		{"MatchNotFound text in message", &RPCError{
			Code:    CodeMethodCallError,
			Message: "MatchNotFound() (EINVAL)",
		}, true},
		{"MatchNotFound but wrong code", &RPCError{
			Code: CodeInvalidParams,
			Data: json.RawMessage(`{"errname":"EINVAL","reason":"MatchNotFound()"}`),
		}, false}, // code is wrong; don't swallow
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsNotFound_CallJobENOENT covers the long-running-job ENOENT
// shape. certificate.delete and pool.delete are CallJob methods -
// the inner job result error is wrapped in a "CallJob X failed:"
// prefix that hides the [ENOENT] from the Data block.
func TestIsNotFound_CallJobENOENT(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"certificate.delete missing id", &RPCError{
			Code:    CodeMethodCallError,
			Message: "CallJob certificate.delete (id=1875) failed: [ENOENT] None: Certificate 53 does not exist",
		}, true},
		{"pool.delete missing id", &RPCError{
			Code:    CodeMethodCallError,
			Message: "CallJob pool.delete (id=42) failed: [ENOENT] None: Pool 7 does not exist",
		}, true},
		{"CallJob with other failure", &RPCError{
			Code:    CodeMethodCallError,
			Message: "CallJob pool.create (id=99) failed: [EIO] disk error",
		}, false}, // missing both ENOENT prefix and 'does not exist'
		{"plain message saying does not exist", &RPCError{
			Code:    CodeMethodCallError,
			Message: "Certificate 53 does not exist",
		}, true}, // catch-all branch 4
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsNotFound_CallJobFailureUnwrapped covers the fallback for the
// CallJob failure path. CallJob wraps job failures with fmt.Errorf
// ("CallJob X (id=N) failed: [ENOENT] ..."), so the inner *RPCError
// is lost, errors.As returns false. The fallback scans the wrapper
// text directly.
func TestIsNotFound_CallJobFailureUnwrapped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"certificate.delete missing id via CallJob wrapper", errors.New("CallJob certificate.delete (id=2274) failed: [ENOENT] None: Certificate 56 does not exist"), true},
		{"deeper wrap with [ENOENT]", errors.New("deleting certificate 56: CallJob certificate.delete (id=2274) failed: [ENOENT] None: Certificate 56 does not exist"), true},
		{"CallJob but not ENOENT", errors.New("CallJob pool.create (id=99) failed: [EIO] disk error"), false},
		{"plain ENOENT without CallJob", errors.New("[ENOENT] foo"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound = %v, want %v", got, tc.want)
			}
		})
	}
}
