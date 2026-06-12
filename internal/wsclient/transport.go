package wsclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// dial establishes the WebSocket connection to wsURL and starts the
// receive loop. Replaces c.conn under connMu and returns nil on
// success. Caller (New() or reconnect()) is responsible for handling
// the post-dial auth handshake.
//
// Any error here is fatal for this connection attempt; reconnect
// logic in reconnect.go decides whether to retry.
func (c *Client) dial(ctx context.Context, wsURL string) error {
	dialCtx, cancel := context.WithTimeout(ctx, c.dialTimeout)
	defer cancel()

	httpClient := &http.Client{}
	if c.insecureSkipVerify {
		// github.com/coder/websocket reuses the *http.Client.Transport for the
		// dial. We can clone DefaultTransport and inject TLS skip here
		// without affecting any other HTTP path in the process.
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional opt-in
		httpClient.Transport = tr
	}

	conn, resp, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		HTTPClient: httpClient,
	})
	// coder/websocket returns the upgrade response object on both success
	// and failure paths, but on a successful 101 upgrade the Body is
	// already hijacked into the conn and resp.Body is nil. On an upgrade
	// failure (non-101 status) Body holds the unconsumed error body and
	// must be closed. Guard on both conditions so the success path is
	// nil-safe and the error path satisfies the bodyclose linter.
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("websocket dial %s: %w", wsURL, err)
	}

	// nhooyr's Dial caps single-message reads at 32KiB by default; some
	// TrueNAS responses (large dataset trees, certificate bundles) can
	// run larger. Bumping to 16 MiB matches the REST client's effective
	// ceiling without introducing a new policy surface.
	conn.SetReadLimit(16 * 1024 * 1024)

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	// recvLoop intentionally outlives the dial context (which is bounded
	// by dialTimeout). The lifetime context c.lifetime is canceled in
	// Close(), which unblocks conn.Read with context.Canceled and lets
	// the loop exit cleanly. lifetime is intentionally not derived from
	// any caller ctx, it is owned by the Client and lasts for the
	// process lifetime of this client.
	go c.recvLoop(c.lifetime, conn) //nolint:gosec,contextcheck // lifetime ctx is owned by Client, not inherited from caller's request ctx
	return nil
}

// recvLoop reads frames off the WebSocket and demuxes by request ID
// into the pending map. Runs until the connection closes for any
// reason, then propagates the failure to every outstanding Call via
// failPending. Reconnect logic (if any) lives in reconnect.go and
// decides whether to dial again.
func (c *Client) recvLoop(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		// Best-effort close; if the conn is already gone, nhooyr returns
		// silently and we still need to propagate to pending callers.
		_ = conn.Close(websocket.StatusInternalError, "recv loop exit")
	}()
	// ctx is c.lifetime, canceled in Close(). conn.Read returns when
	// either the conn drops on its own or ctx is canceled, and the
	// post-Read isClosing check catches the deliberate-close case.
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			// Distinguish deliberate close (Close()) from unexpected
			// drop. The former drains pending via Close() already; the
			// latter needs failPending here.
			if c.isClosing() {
				return
			}
			// Mark the conn as dead so reconnectIfNeeded actually
			// redials on the next idempotent retry. Without this
			// clear, c.conn still references the (closed) conn and
			// reconnectIfNeeded reports it healthy on entry.
			c.connMu.Lock()
			if c.conn == conn {
				c.conn = nil
			}
			c.connMu.Unlock()
			c.failPending(fmt.Errorf("%w: %w", ErrConnectionLost, err))
			return
		}

		var resp rpcResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			// A malformed frame from the server is a bug. Log it and
			// keep reading, abandoning the connection on a single bad
			// frame would tear down every other in-flight call.
			//
			// We cannot route this anywhere because we cannot extract
			// an ID from a frame that did not parse, so this is
			// effectively a "lost" response that the originating
			// Call() will surface via its own timeout.
			continue
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[resp.ID]
		if ok {
			delete(c.pending, resp.ID)
		}
		c.pendingMu.Unlock()

		if !ok {
			// Late or unknown response (the originating Call already
			// timed out, or the server emitted a notification we don't
			// subscribe to). Discard.
			continue
		}

		// The channel is fresh per-call (cap=1) and we delete the
		// pending entry above before sending, so duplicate or stale
		// IDs land in the late-response branch and never reach here.
		// That guarantees this send will not block.
		ch <- &resp
		close(ch)
	}
}

// isClosing reports whether the Client is shutting down.
func (c *Client) isClosing() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

// marshalRequest is the JSON marshaler for outgoing rpcRequest frames.
// Defaults to json.Marshal; tests can swap to inject a marshal failure
// and exercise the err path in sendFrame.
var marshalRequest = json.Marshal

// sendFrame marshals req to JSON and writes it to the active
// connection. Returns ErrConnectionLost wrapping any transport error.
// Holds connMu only for the duration of the write so concurrent
// senders serialize but a slow write does not block the read loop.
func (c *Client) sendFrame(ctx context.Context, req rpcRequest) error {
	data, err := marshalRequest(req)
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}

	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	if conn == nil {
		return ErrConnectionLost
	}

	return wrapWriteErr(conn.Write(ctx, websocket.MessageText, data))
}

// wrapWriteErr classifies a conn.Write error: context cancellation
// passes through unmodified so callers can errors.Is() against the
// context sentinels; everything else is wrapped as ErrConnectionLost.
// Extracted so the rare ctx-error branch is unit-testable without
// staging a race against the kernel buffer.
func wrapWriteErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrConnectionLost, err)
}

// backoffDelay returns the delay before retry attempt n. Lifted from
// client/client.go:233-255 so the wsclient retry envelope behaves
// identically to the REST client's. Pure function, safe to cover via
// table-driven test.
func backoffDelay(p RetryPolicy, attempt int) time.Duration {
	base := p.BaseDelay
	if base <= 0 {
		base = 500 * time.Millisecond
	}
	maxD := p.MaxDelay
	if maxD <= 0 {
		maxD = 10 * time.Second
	}
	if attempt < 0 {
		attempt = 0
	}
	shift := attempt
	if shift > 20 {
		shift = 20
	}
	d := base << shift //nolint:gosec // bounded by cap
	if d <= 0 || d > maxD {
		d = maxD
	}
	return d
}

// sleepCtx sleeps for d, returning early if ctx is canceled. Mirror of
// client/client.go's helper.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
