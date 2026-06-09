package wsclient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// DefaultRequestTimeout is the per-call deadline used when a CallOptions
// does not override. Matches client.DefaultRequestTimeout so operators
// migrating between the two transports do not see latency surprises.
const DefaultRequestTimeout = 60 * time.Second

// DefaultDialTimeout bounds the WebSocket handshake. Distinct from the
// per-call timeout because dial is a one-time cost paid at provider
// Configure; making it too short risks Configure failing on a slow
// link, too long delays useful diagnostics.
const DefaultDialTimeout = 30 * time.Second

// RetryPolicy mirrors client.RetryPolicy so configuration plumbing on
// the provider side is identical between transports.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of TOTAL attempts including
	// the first. A value of 1 disables retry; 0 falls back to the
	// default in DefaultRetryPolicy.
	MaxAttempts int
	// BaseDelay is the first-retry delay. Subsequent retries multiply
	// it by 2^attempt, capped at MaxDelay.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff.
	MaxDelay time.Duration
}

// DefaultRetryPolicy is the out-of-the-box retry policy. Identical to
// the REST client's defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
	}
}

// CallOptions configure a single Call. Zero value is the safe default:
// no retry on connection loss, treat as mutating, run to completion
// rather than as a job.
type CallOptions struct {
	// Idempotent, when true, allows the transport to retry the call
	// after a connection drop. The default is false: the transport
	// will surface ErrConnectionLost to the caller, who must decide
	// whether replaying the call is safe. Per the Phase 0 design
	// decision, this is opt-in, not opt-out.
	Idempotent bool
	// Job, when true, treats the call as TrueNAS' long-running "job"
	// pattern: the server returns a job ID immediately and the client
	// polls core.get_jobs until the job reaches a terminal state.
	// Equivalent to the REST helper's `WaitForJob` flow.
	Job bool
	// Timeout overrides DefaultRequestTimeout for this call. Zero
	// means use the default.
	Timeout time.Duration
	// Read marks a method as read-only even if the suffix-based
	// classification would call it mutating. Used for unusual methods
	// like core.ping.
	Read bool
	// Mutates forces mutating classification even if the suffix would
	// classify it as read-only. Used for the rare server method that
	// mutates state under a "query"-style name.
	Mutates bool
	// Destroys forces destructive classification regardless of name
	// suffix. Used for methods that destroy state under non-canonical
	// names (e.g. system.reboot).
	Destroys bool
	// disableReconnect, when true, suppresses the retry loop's
	// reconnectIfNeeded branch. Internal-only — set by authenticate()
	// when called from inside reconnectIfNeeded so a transport drop
	// during the auth handshake surfaces immediately rather than
	// recursively re-entering the reconnect path (which would deadlock
	// on the package-level reconnectMu).
	disableReconnect bool
}

// Client handles communication with the TrueNAS SCALE JSON-RPC 2.0
// WebSocket API (TrueNAS 25.04 and later). One Client wraps a single
// persistent WebSocket connection with multiplexed request/response
// correlation, automatic reconnect, and the same safety rails the
// legacy REST client exposes.
//
// A *Client is safe for concurrent use. Call() takes a context and
// returns a JSON-RPC result body; per-resource helpers (GetDataset,
// CreateShareNFS, etc.) wrap Call with typed parameters and result
// unmarshaling so resource code never touches raw JSON.
type Client struct {
	// baseURL is the http(s) URL of the TrueNAS instance. Stored in
	// the original https:// form for log readability; the WebSocket
	// dial path translates to ws(s):// before connecting.
	baseURL string
	apiKey  string

	// conn is the active nhooyr WebSocket. Replaced on reconnect.
	// Access only with connMu held.
	conn   *websocket.Conn
	connMu sync.Mutex

	// nextID is a monotonic counter for JSON-RPC request IDs. Atomic
	// because Call() is concurrent.
	nextID atomic.Uint64

	// pending maps an outstanding request ID to the channel that the
	// receive loop posts the response into. Access only with
	// pendingMu held.
	pending   map[uint64]chan *rpcResponse
	pendingMu sync.Mutex

	// closed is closed when the Client is shutting down. Used by the
	// receive loop and Call() to short-circuit further work.
	closed   chan struct{}
	closeMu  sync.Mutex
	isClosed bool

	// lifetime is the context tied to the Client's life — created in
	// New(), canceled in Close(). Passed to recvLoop so its conn.Read
	// can be unblocked by Close(). Distinct from any request-scoped
	// context the Configure caller passes in.
	lifetime       context.Context
	lifetimeCancel context.CancelFunc

	// dialTimeout, requestTimeout mirror the REST client.
	dialTimeout    time.Duration
	requestTimeout time.Duration

	// RetryPolicy is the public knob test code historically poked
	// to assert retry behaviour against the REST client. Kept
	// exported for test-suite compatibility during the v2.0
	// WebSocket cutover; the wsclient retry path consults this
	// field exactly like the REST client did.
	RetryPolicy RetryPolicy

	// ReadOnly and DestroyProtection are exported safety rails;
	// see readonly.go and destroy_protection.go.
	ReadOnly          bool
	DestroyProtection bool

	// insecureSkipVerify wires through to the WebSocket dialer's TLS
	// config. Same opt-in flag as the REST client; surfaces in the
	// provider schema as `insecure`.
	insecureSkipVerify bool
}

// rpcRequest is the JSON-RPC 2.0 request envelope sent over the wire.
type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint64        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// rpcResponse is the JSON-RPC 2.0 response envelope received from the
// server. Exactly one of Result or Error is populated on a well-formed
// response. Result is held as RawMessage so callers can decode the
// inner value into the per-method typed result struct.
//
// transportErr is internal-only — it carries a synthesized error from
// failPending() that needs to surface up to call.go's retry loop with
// its error chain intact (so errors.Is(err, ErrConnectionLost) still
// matches). Wire responses always have transportErr nil.
type rpcResponse struct {
	JSONRPC      string          `json:"jsonrpc"`
	ID           uint64          `json:"id"`
	Result       json.RawMessage `json:"result,omitempty"`
	Error        *RPCError       `json:"error,omitempty"`
	transportErr error
}

// NewWithOptions is a compatibility shim matching the legacy
// client.NewWithOptions signature so tests that historically built a
// REST *client.Client can transparently build a *wsclient.Client
// instead during the v2.0 cutover. Uses a background context for the
// dial+auth handshake; for production code that has a Configure ctx,
// prefer New(ctx, ...).
func NewWithOptions(baseURL, apiKey string, insecureSkipVerify bool) (*Client, error) {
	return New(context.Background(), baseURL, apiKey, insecureSkipVerify)
}

// New constructs a wsclient.Client connected to baseURL using apiKey.
// Returns the connected, authenticated client on success or an error
// describing the failure (network, TLS, auth, version).
func New(ctx context.Context, baseURL, apiKey string, insecureSkipVerify bool) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("truenas base URL is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("truenas API key is required")
	}

	wsURL, err := wsURLFromBase(baseURL)
	if err != nil {
		return nil, err
	}

	lifetime, cancel := context.WithCancel(context.Background())
	c := &Client{
		baseURL:            baseURL,
		apiKey:             apiKey,
		pending:            make(map[uint64]chan *rpcResponse),
		closed:             make(chan struct{}),
		lifetime:           lifetime,
		lifetimeCancel:     cancel,
		dialTimeout:        DefaultDialTimeout,
		requestTimeout:     DefaultRequestTimeout,
		RetryPolicy:        DefaultRetryPolicy(),
		insecureSkipVerify: insecureSkipVerify,
	}

	if err := c.dial(ctx, wsURL); err != nil {
		return nil, err
	}

	if err := c.authenticate(ctx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	return c, nil
}

// RequestTimeout returns the per-call request timeout currently in
// effect. Mirror of *client.Client.RequestTimeout() so callers can
// log the configured value without caring which transport is wired.
func (c *Client) RequestTimeout() time.Duration {
	return c.requestTimeout
}

// SetRequestTimeout updates the per-call request timeout. A zero or
// negative duration falls back to DefaultRequestTimeout.
func (c *Client) SetRequestTimeout(d time.Duration) {
	if d <= 0 {
		d = DefaultRequestTimeout
	}
	c.requestTimeout = d
}

// SetRetryPolicy overrides the retry policy. Zero-valued fields fall
// back to DefaultRetryPolicy.
func (c *Client) SetRetryPolicy(p RetryPolicy) {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = DefaultRetryPolicy().MaxAttempts
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = DefaultRetryPolicy().BaseDelay
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = DefaultRetryPolicy().MaxDelay
	}
	c.RetryPolicy = p
}

// Close shuts down the WebSocket connection and signals every in-flight
// Call to return ErrShuttingDown. Safe to call multiple times. Always
// returns nil — the Close error from nhooyr is ignored because by the
// time we are calling it, we no longer care.
func (c *Client) Close() error {
	c.closeMu.Lock()
	if c.isClosed {
		c.closeMu.Unlock()
		return nil
	}
	c.isClosed = true
	close(c.closed)
	if c.lifetimeCancel != nil {
		c.lifetimeCancel()
	}
	c.closeMu.Unlock()

	c.connMu.Lock()
	conn := c.conn
	c.conn = nil
	c.connMu.Unlock()

	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "client closed")
	}

	c.failPending(ErrShuttingDown)
	return nil
}

// failPending closes every pending response channel with err. Called
// from Close() and from the receive loop on connection drop. Each
// pending channel receives a synthetic rpcResponse whose transportErr
// field carries err with its error chain intact, so call.go's retry
// loop can errors.Is() against ErrConnectionLost / ErrShuttingDown
// and decide whether to retry. We deliberately do NOT wrap err inside
// an *RPCError — that flattens the chain and breaks the reconnect
// contract for in-flight idempotent calls.
func (c *Client) failPending(err error) {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	for id, ch := range c.pending {
		select {
		case ch <- &rpcResponse{ID: id, transportErr: err}:
		default:
		}
		close(ch)
		delete(c.pending, id)
	}
}

// nextRequestID returns a fresh monotonic ID for a JSON-RPC request.
func (c *Client) nextRequestID() uint64 {
	return c.nextID.Add(1)
}

// newCorrelationID returns a 16-char lowercase hex correlation ID,
// threaded through tflog lines for a single Call. Distinct from the
// JSON-RPC request ID (which is uint64 and round-tripped to the
// server) — this one is for log correlation only.
func newCorrelationID() string {
	return newCorrelationIDFrom(rand.Reader)
}

// newCorrelationIDFrom is the unit-testable form of newCorrelationID.
func newCorrelationIDFrom(src io.Reader) string {
	var b [8]byte
	if _, err := io.ReadFull(src, b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

// wsURLFromBase converts an http(s) base URL to the ws(s)://host/api/current
// endpoint that TrueNAS exposes for its JSON-RPC 2.0 transport. Strips
// trailing slashes and any /api/v2.0 suffix the operator may have
// carried over from REST configuration. Returns an error on a
// malformed URL.
func wsURLFromBase(base string) (string, error) {
	if base == "" {
		return "", errors.New("base URL is empty")
	}
	trimmed := strings.TrimRight(base, "/")
	trimmed = strings.TrimSuffix(trimmed, "/api/v2.0")
	switch {
	case strings.HasPrefix(trimmed, "https://"):
		return "wss://" + strings.TrimPrefix(trimmed, "https://") + "/api/current", nil
	case strings.HasPrefix(trimmed, "http://"):
		return "ws://" + strings.TrimPrefix(trimmed, "http://") + "/api/current", nil
	case strings.HasPrefix(trimmed, "wss://"), strings.HasPrefix(trimmed, "ws://"):
		// Already a WebSocket URL; trust the operator and strip any
		// stale /api/v2.0 suffix as above.
		return trimmed + "/api/current", nil
	default:
		return "", fmt.Errorf("unsupported URL scheme in %q (expected http://, https://, ws:// or wss://)", base)
	}
}
