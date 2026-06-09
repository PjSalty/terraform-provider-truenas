package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Call sends a JSON-RPC request and blocks until the matching response
// arrives, the per-call deadline elapses, or the connection drops. The
// returned RawMessage is the un-decoded `result` field from the
// server's response, suitable for json.Unmarshal into a per-method
// typed struct.
//
// Safety rails (read-only / destroy-protection) are checked BEFORE
// the request hits the wire. Retry-on-disconnect is opt-in via
// opts.Idempotent and only applies when ErrConnectionLost surfaces; a
// well-formed RPCError is returned to the caller without retry,
// because re-sending a server-level reject would change nothing.
//
// The correlation ID threaded through tflog is distinct from the
// JSON-RPC `id`: the JSON-RPC id is uint64 and tied to demux, the
// correlation ID is hex and tied to operator log readability.
func (c *Client) Call(ctx context.Context, method string, params []interface{}, opts CallOptions) (json.RawMessage, error) {
	if c == nil {
		return nil, errors.New("nil wsclient.Client")
	}
	if method == "" {
		return nil, errors.New("method name is required")
	}

	if err := c.checkReadOnly(method, opts); err != nil {
		return nil, err
	}
	if err := c.checkDestroyProtection(method, opts); err != nil {
		return nil, err
	}

	if c.isClosing() {
		return nil, ErrShuttingDown
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = c.requestTimeout
	}

	correlationID := newCorrelationID()
	logCtx := tflog.SetField(ctx, "ws_correlation_id", correlationID)
	logCtx = tflog.SetField(logCtx, "ws_method", method)

	policy := c.RetryPolicy
	if !opts.Idempotent {
		// Non-idempotent calls get exactly one attempt: a transport
		// drop must surface to the caller for explicit replay
		// decisions. The safety rails check above already fired, so
		// this only affects the wire path.
		policy.MaxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := backoffDelay(policy, attempt-1)
			tflog.Debug(logCtx, "wsclient retry", map[string]interface{}{
				"attempt": attempt,
				"delay":   delay.String(),
				"reason":  lastErr.Error(),
			})
			if err := sleepCtx(ctx, delay); err != nil {
				return nil, err
			}
		}

		result, err := c.callOnce(logCtx, method, params, timeout)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Only ErrConnectionLost is retried; well-formed RPCErrors and
		// context cancellation surface immediately. Before retrying,
		// trigger a reconnect so the next callOnce dials a fresh
		// transport rather than re-using a dead conn.
		//
		// The disableReconnect escape hatch suppresses this branch for
		// the auth-handshake call issued from inside reconnectIfNeeded:
		// re-entering reconnect from there would deadlock on the
		// package-level reconnectMu. The auth call surfaces the
		// connection-lost error to its caller (reconnectIfNeeded),
		// which loops on its own dial-retry budget instead.
		if errors.Is(err, ErrConnectionLost) && !opts.disableReconnect {
			if rerr := c.reconnectIfNeeded(ctx); rerr != nil {
				return nil, rerr
			}
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

// callOnce performs a single JSON-RPC request/response round trip
// against the active WebSocket connection. Registers the request in
// the pending map, sends the frame, and blocks on the response
// channel or the call deadline.
func (c *Client) callOnce(ctx context.Context, method string, params []interface{}, timeout time.Duration) (json.RawMessage, error) {
	id := c.nextRequestID()
	respCh := make(chan *rpcResponse, 1)

	c.pendingMu.Lock()
	c.pending[id] = respCh
	c.pendingMu.Unlock()

	cleanup := func() {
		c.pendingMu.Lock()
		if ch, ok := c.pending[id]; ok && ch == respCh {
			delete(c.pending, id)
		}
		c.pendingMu.Unlock()
	}

	if params == nil {
		// JSON-RPC requires the params field to exist; encode as an
		// empty array rather than null so the server's strict parser
		// accepts the request.
		params = []interface{}{}
	}
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	sendCtx, sendCancel := context.WithTimeout(ctx, timeout)
	defer sendCancel()
	if err := c.sendFrame(sendCtx, req); err != nil {
		cleanup()
		return nil, err
	}

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	select {
	case resp := <-respCh:
		// recvLoop sends a non-nil *rpcResponse before closing ch, and
		// failPending sends a synthetic value before closing. Both
		// paths guarantee resp != nil when the channel produces.
		//
		// Synthesized transport errors come through resp.transportErr
		// with their error chain intact; we surface those as-is so
		// the retry loop can match errors.Is(err, ErrConnectionLost).
		if resp.transportErr != nil {
			return nil, resp.transportErr
		}
		if resp.Error != nil {
			// Run the message through redactMessage before returning
			// so a verbose middlewared error cannot bleed sensitive
			// values through tflog or Diagnostics.
			scrubbed := *resp.Error
			scrubbed.Message = redactMessage(resp.Error.Message)
			scrubbed.Data = redactJSONBody(resp.Error.Data)
			return nil, &scrubbed
		}
		return resp.Result, nil

	case <-deadline.C:
		cleanup()
		return nil, fmt.Errorf("wsclient: call %s timed out after %s", method, timeout)

	case <-ctx.Done():
		cleanup()
		return nil, ctx.Err()

	case <-c.closed:
		cleanup()
		return nil, ErrShuttingDown
	}
}
