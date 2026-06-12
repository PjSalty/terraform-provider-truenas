package wsclient

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// reconnectMu serializes reconnect attempts so two goroutines that both
// observe ErrConnectionLost do not redial the server in parallel. The
// first holder runs the dial+auth handshake; subsequent holders see
// the conn already healthy on entry and return.
var reconnectMu sync.Mutex // package-level, Client field would be one mutex per Client instance, fine

// reconnectIfNeeded checks whether c.conn is alive and, if not,
// performs a single redial+auth sequence. Returns nil when the conn
// is healthy on return (either it was, or this call brought it back),
// and a wrapped error if the redial failed.
//
// Designed for use from Call() retry loops: an idempotent call that
// hits ErrConnectionLost can wrap reconnectIfNeeded around its retry
// to bring the conn back before re-issuing the request. Non-idempotent
// callers MUST NOT use this, replaying a non-idempotent call after
// reconnect could double-execute the operation server-side.
//
// reconnectIfNeeded blocks for at most c.dialTimeout. If the redial
// runs into transient failures it backs off using c.RetryPolicy and
// re-tries within the window; if it cannot reconnect before the
// deadline, it returns ErrConnectionLost.
func (c *Client) reconnectIfNeeded(ctx context.Context) error {
	if c.isClosing() {
		return ErrShuttingDown
	}

	c.connMu.Lock()
	healthy := c.conn != nil
	c.connMu.Unlock()
	if healthy {
		return nil
	}

	reconnectMu.Lock()
	defer reconnectMu.Unlock()

	// Re-check after acquiring the mutex: another goroutine may have
	// reconnected while we were parked.
	c.connMu.Lock()
	healthy = c.conn != nil
	c.connMu.Unlock()
	if healthy {
		return nil
	}

	tflog.Info(ctx, "wsclient: reconnecting after transport drop", map[string]interface{}{
		"url": c.baseURL,
	})

	wsURL, err := wsURLFromBase(c.baseURL)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(c.dialTimeout)
	for attempt := 0; ; attempt++ {
		// Drop any pending callers, their replies were going to a
		// dead conn and will never arrive. Idempotent retriers will
		// re-issue against the fresh conn after this returns.
		c.failPending(ErrConnectionLost)

		dialErr := c.dial(ctx, wsURL)
		if dialErr == nil {
			authErr := c.authenticate(ctx)
			if authErr == nil {
				tflog.Info(ctx, "wsclient: reconnected", nil)
				return nil
			}
			dialErr = authErr
		}

		tflog.Warn(ctx, "wsclient: reconnect attempt failed", map[string]interface{}{
			"attempt": attempt,
			"error":   dialErr.Error(),
		})

		if errors.Is(dialErr, context.Canceled) || errors.Is(dialErr, context.DeadlineExceeded) {
			return dialErr
		}
		if time.Now().After(deadline) {
			return ErrConnectionLost
		}

		if err := sleepCtx(ctx, backoffDelay(c.RetryPolicy, attempt)); err != nil {
			return err
		}
	}
}
