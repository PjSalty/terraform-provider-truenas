package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// authJitterDelay returns a uniformly-jittered backoff in the range
// [base, base*3) capped at maxDelay. Used by authenticate() to spread
// concurrent retries off the rate-limit window when multiple acc
// tests dial the same TrueNAS simultaneously.
func authJitterDelay(base, maxDelay time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	// rand.Float64 ∈ [0, 1); scale into [1, 3) so the jittered delay
	// straddles `base` rather than just adding a fraction on top.
	d := time.Duration(float64(base) * (1 + rand.Float64()*2)) //nolint:gosec // jitter doesn't need crypto rand
	if d > maxDelay {
		return maxDelay
	}
	return d
}

// authenticate performs the post-dial JSON-RPC handshake that turns an
// anonymous WebSocket into an authenticated session. TrueNAS exposes
// `auth.login_with_api_key` which takes the literal API key string in
// the params array. On success the server returns true; on failure
// either a typed RPCError or a `false` result.
//
// The handshake is non-idempotent (a successful login changes server
// session state), so we do not flag it Idempotent. It also is not
// classified as mutating for the read-only gate's purposes — a read-
// only client still needs to authenticate before it can read.
func (c *Client) authenticate(ctx context.Context) error {
	// TrueNAS' authentication middleware throttles rapid-fire auth
	// calls; the acceptance suite hammers the endpoint as each
	// per-test provider.Configure() rebuilds a fresh client. The
	// server surfaces this as [EBUSY] Rate Limit Exceeded under
	// CodeMethodCallError. Back off with capped exponential delay
	// plus per-attempt jitter so when N concurrent acc tests hit
	// the gate simultaneously they don't redial in lockstep.
	//
	// The 12-attempt budget + 6s ceiling gives ~30s of total wait
	// under worst-case retry, which is comfortably under the
	// Configure ctx deadline (5 min default) and the framework's
	// per-step deadline.
	const (
		maxAttempts = 12
		maxDelay    = 6 * time.Second
	)
	delay := 200 * time.Millisecond
	var result json.RawMessage
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err = c.Call(ctx, "auth.login_with_api_key", []interface{}{c.apiKey}, CallOptions{
			Read:             true, // bypass read-only suffix classifier; auth must succeed even with ReadOnly=true
			disableReconnect: true, // reconnectIfNeeded calls authenticate; re-entering reconnect would deadlock on reconnectMu
		})
		if err == nil {
			break
		}
		if !isAuthRateLimited(err) || attempt == maxAttempts {
			break
		}
		// Decorrelated jitter: pick a delay uniformly from
		// [base, base * 3) capped at maxDelay. Spreads concurrent
		// retries so the next round doesn't pile back into the
		// rate limit window in lockstep.
		jittered := authJitterDelay(delay, maxDelay)
		select {
		case <-ctx.Done():
			return fmt.Errorf("authenticate: %w (last error: %w)", ctx.Err(), err)
		case <-time.After(jittered):
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
	if err != nil {
		return err
	}
	var ok bool
	if err := json.Unmarshal(result, &ok); err != nil {
		// TrueNAS 25.04+ has been observed returning either a bare
		// boolean or an object with a `username` field. Accept either
		// shape; reject anything else as a hard failure.
		var obj map[string]interface{}
		if jsonErr := json.Unmarshal(result, &obj); jsonErr != nil {
			return fmt.Errorf("auth.login_with_api_key: unexpected result shape: %s", string(result))
		}
		return nil
	}
	if !ok {
		return errors.New("auth.login_with_api_key: server returned false (invalid API key?)")
	}
	return nil
}

// isAuthRateLimited reports whether err carries TrueNAS'
// "[EBUSY] Rate Limit Exceeded" signature. We accept both the
// CodeMethodCallError class with errname=EBUSY (seen on 25.04) and a
// plain message-text scan as a fallback (some middlewared versions
// don't populate the data block on rate-limit refusals).
func isAuthRateLimited(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	if rpcErr.Code != CodeMethodCallError {
		return false
	}
	errname, reason := rpcErr.errnameAndReason()
	if errname == "EBUSY" {
		return true
	}
	low := strings.ToLower(reason + " " + rpcErr.Message)
	return strings.Contains(low, "rate limit")
}
