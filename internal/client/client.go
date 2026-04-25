package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// newRequestID returns a fresh 16-char lowercase hex ID; one per logical
// API call. Generated from crypto/rand for negligible collision risk
// across a provider run and zero dependence on global UUID state. Used
// as the X-Request-ID header and threaded through every tflog line for
// this request so operator traces can correlate client-side breadcrumbs
// with TrueNAS middlewared audit log entries.
func newRequestID() string {
	return newRequestIDFrom(rand.Reader)
}

// newRequestIDFrom is the unit-testable form of newRequestID. It takes
// any io.Reader as its entropy source so tests can inject a failing
// reader and exercise the fallback branch (which would otherwise be
// impossible to hit — crypto/rand.Reader never fails on a healthy
// system). newRequestID() is a thin wrapper that passes rand.Reader.
func newRequestIDFrom(src io.Reader) string {
	var b [8]byte
	if _, err := io.ReadFull(src, b[:]); err != nil {
		// crypto/rand failing means the kernel's entropy source is dead,
		// which is a catastrophic environment state — fall back to a
		// timestamp so we still emit *something* traceable and don't
		// return an empty ID that would silently drop the header.
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

// RetryPolicy configures automatic retry behavior for idempotent HTTP requests.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryPolicy is the out-of-the-box retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
	}
}

// Client handles communication with the TrueNAS SCALE REST API v2.0.
type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	RetryPolicy RetryPolicy
	// ReadOnly, when true, causes every mutating request (anything other
	// than GET/HEAD/OPTIONS) to fail with ErrReadOnly BEFORE a network
	// call is made. Used as a prod safety rail during phased rollout:
	// point the provider at a production TrueNAS, run `terraform plan`,
	// and be physically unable to mutate anything. See readonly.go.
	ReadOnly bool
	// DestroyProtection, when true, blocks ONLY DELETE requests at the
	// client layer (with ErrDestroyProtected) while allowing GET, POST,
	// and PUT through. This is the apply-safe rail: the first production
	// apply can create and update resources but cannot delete anything
	// until the operator explicitly opts in by unsetting the flag or
	// the TRUENAS_DESTROY_PROTECTION env var. See destroy_protection.go.
	DestroyProtection bool
}

// DefaultRequestTimeout is the per-request HTTP timeout applied when the
// caller does not configure a custom value. 60s is enough for every
// TrueNAS API endpoint on a healthy system but may be too tight for
// pool scrub status, large dataset listings, or replication checks on
// loaded production systems. Callers that hit those paths should
// raise the timeout via SetRequestTimeout or the provider schema.
const DefaultRequestTimeout = 60 * time.Second

// New creates a new TrueNAS API client with the default request timeout.
func New(baseURL, apiKey string) (*Client, error) {
	return NewWithOptions(baseURL, apiKey, false)
}

// NewWithOptions creates a new TrueNAS API client with optional TLS verification skip.
func NewWithOptions(baseURL, apiKey string, insecureSkipVerify bool) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("truenas base URL is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("truenas API key is required")
	}

	// Normalize base URL
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api/v2.0") {
		baseURL += "/api/v2.0"
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional opt-in
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   DefaultRequestTimeout,
			Transport: transport,
		},
		RetryPolicy: DefaultRetryPolicy(),
	}, nil
}

// SetRequestTimeout updates the per-request HTTP timeout. A duration of
// zero or less is ignored so callers cannot accidentally disable the
// safety rail by reading a malformed config value. Intended to be
// called once, during provider Configure, before any resource CRUD.
func (c *Client) SetRequestTimeout(d time.Duration) {
	if c == nil || c.httpClient == nil || d <= 0 {
		return
	}
	c.httpClient.Timeout = d
}

// RequestTimeout returns the current per-request HTTP timeout. Exists
// primarily for tests and observability; production code should rarely
// need to read it.
func (c *Client) RequestTimeout() time.Duration {
	if c == nil || c.httpClient == nil {
		return 0
	}
	return c.httpClient.Timeout
}

// IsNotFound reports whether err indicates the target resource was not found.
// It unwraps wrapped errors and recognizes:
//   - 404 HTTP responses
//   - TrueNAS's 422 "Object with id=X does not exist" / "does not exist" bodies
//   - Job failures containing [ENOENT] or "does not exist"
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusNotFound {
			return true
		}
		if apiErr.StatusCode == http.StatusUnprocessableEntity {
			lower := strings.ToLower(apiErr.Body + " " + apiErr.Message)
			if strings.Contains(lower, "does not exist") {
				return true
			}
		}
		return false
	}
	// Non-APIError — check the text for ENOENT/not-found markers commonly
	// produced by TrueNAS middleware job failures.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "[enoent]") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found") {
		return true
	}
	return false
}

// isRetryableTransportErr returns true for transport-level errors that are
// safe to retry (connection resets, broken pipes, EOF, timeouts).
func isRetryableTransportErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return true
	}
	msg := err.Error()
	for _, needle := range []string{"connection reset", "connection refused", "broken pipe", "EOF"} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// parseRetryAfter extracts a delay from a Retry-After response header.
// Accepts both delta-seconds and HTTP-date forms. Returns 0 if absent
// or unparseable.
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	h := resp.Header.Get("Retry-After")
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// backoffDelay returns the delay before the next retry attempt, using
// exponential backoff capped at p.MaxDelay.
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
	// Cap the shift to avoid overflow.
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

// sleepCtx sleeps for d but returns early if ctx is canceled.
// A zero or negative duration returns ctx.Err() immediately.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// isIdempotent reports whether the HTTP method is safe to retry.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return false
}

// shouldRetryStatus reports whether the given status code warrants a retry.
func shouldRetryStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// APIError represents an error response from the TrueNAS API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
	// retryAfter is set internally when a retry should honor a server-provided delay.
	retryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("TrueNAS API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("TrueNAS API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// doRequest executes an HTTP request against the TrueNAS API with retry support.
// Idempotent methods (GET/HEAD/PUT/DELETE/OPTIONS) are retried on transient
// 5xx/429 responses and transport-level failures. Non-idempotent POST requests
// are NOT retried automatically to avoid side-effect duplication.
//
// Every call is tagged with a freshly-generated request ID (16 hex chars)
// that is set as the outgoing `X-Request-ID` header AND threaded through
// every structured log line, so operators can correlate client-side
// breadcrumbs with TrueNAS middlewared audit entries.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	reqID := newRequestID()
	tflog.Trace(ctx, "client doRequest start", map[string]any{
		"method": method, "path": path, "req_id": reqID,
	})

	// Read-only safety rail: refuse every mutating method (POST/PUT/DELETE)
	// before any network call is made. This takes effect BEFORE marshaling
	// the body, BEFORE contacting the server, and BEFORE the retry loop —
	// the target system cannot observe even a dropped attempt.
	if err := c.checkReadOnly(method, path); err != nil {
		tflog.Warn(ctx, "read-only client refused mutating request", map[string]any{
			"method": method, "path": path, "req_id": reqID,
		})
		return nil, err
	}
	// Destroy-protection safety rail: refuse DELETE specifically. Runs
	// AFTER the read-only check so a client with both flags set still
	// reports the clearer ReadOnly error (which is strictly broader)
	// rather than the DestroyProtected error.
	if err := c.checkDestroyProtection(method, path); err != nil {
		tflog.Warn(ctx, "destroy-protected client refused DELETE", map[string]any{
			"method": method, "path": path, "req_id": reqID,
		})
		return nil, err
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	maxAttempts := c.RetryPolicy.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	idempotent := isIdempotent(method)

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Determine backoff. If the last error was an APIError with
			// Retry-After semantics, respect that.
			delay := backoffDelay(c.RetryPolicy, attempt-1)
			var apiErr *APIError
			if errors.As(lastErr, &apiErr) && apiErr.retryAfter > 0 {
				delay = apiErr.retryAfter
			}
			if err := sleepCtx(ctx, delay); err != nil {
				return nil, err
			}
		}

		respBody, err := c.doOnce(ctx, method, path, bodyBytes, reqID)
		if err == nil {
			tflog.Trace(ctx, "client doRequest success", map[string]any{
				"method": method, "path": path, "attempts": attempt + 1, "req_id": reqID,
			})
			return respBody, nil
		}
		lastErr = err

		// Classify the error:
		//   - APIError: HTTP-level failure. doOnce has already stamped
		//     retryAfter on it from the response headers before the
		//     body was closed. Retryable only for idempotent methods
		//     AND retryable status codes.
		//   - non-APIError: transport-level or body-read failure.
		//     Retryable for both idempotent and non-idempotent methods
		//     (since the request may not have reached the server) IF
		//     isRetryableTransportErr.
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			if !idempotent {
				return nil, err
			}
			if !shouldRetryStatus(apiErr.StatusCode) {
				return nil, err
			}
			continue
		}
		// Transport-level or body-read error — safe to retry if the
		// error class allows, regardless of method idempotency.
		if !isRetryableTransportErr(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

// doOnce performs a single HTTP attempt. Returns the response body on
// success, or an error wrapping an APIError for HTTP errors / raw
// error for transport failures. The *http.Response is fully consumed
// and closed before return — callers receive bytes, never a
// still-open response — which is how we guarantee bodyclose safety at
// the caller boundary regardless of retry logic.
//
// If the HTTP response has a Retry-After header, doOnce parses it and
// stamps the resulting duration onto the returned APIError, so the
// retry loop in doRequest can honor it without holding a reference to
// the response object.
//
// The reqID parameter is the logical-operation correlation ID generated
// by doRequest; it is passed through on EVERY retry so operators can
// identify all attempts belonging to the same caller intent. doOnce sets
// it on the outgoing X-Request-ID header.
func (c *Client) doOnce(ctx context.Context, method, path string, bodyBytes []byte, reqID string) ([]byte, error) {
	var reqBody io.Reader
	if bodyBytes != nil {
		reqBody = bytes.NewReader(bodyBytes)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request to %s: %w", reqURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Redact sensitive fields before they land on APIError.Body, which
		// flows through err.Error() into Diagnostics and ultimately into
		// terraform's plain-text output and state-file error fields. Phase G.
		safeBody := redactJSONBody(respBody)
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(safeBody),
			retryAfter: parseRetryAfter(resp),
		}
		// Try to parse error message from JSON response
		var errResp map[string]interface{}
		if json.Unmarshal(respBody, &errResp) == nil {
			if msg, ok := errResp["message"].(string); ok {
				apiErr.Message = redactMessage(msg)
			}
		}
		return nil, apiErr
	}

	return respBody, nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil)
}

// DeleteWithBody performs a DELETE request with a JSON body.
func (c *Client) DeleteWithBody(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, body)
}

// --- Dataset API ---

// PropertyValue represents a ZFS property with a value field.
type PropertyValue struct {
	Value  string      `json:"value"`
	Source string      `json:"source"`
	Parsed interface{} `json:"parsed"`
}

// PropertyRawVal represents a ZFS property with a rawvalue field.
type PropertyRawVal struct {
	Value    string      `json:"value"`
	Rawvalue string      `json:"rawvalue"`
	Source   string      `json:"source"`
	Parsed   interface{} `json:"parsed"`
}

// --- NFS Share API ---

// --- SMB Share API ---

// --- Snapshot Task API ---

// Schedule represents a cron-like schedule.
type Schedule struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
}

// --- iSCSI Target API ---

// --- iSCSI Portal API ---

// --- iSCSI Extent API ---

// --- iSCSI Initiator API ---

// --- CronJob API ---

// --- Pool API (read-only) ---

// --- System Info API (read-only) ---

// --- Alert Service API ---

// --- Replication API ---

// --- Zvol API (uses dataset endpoint with type=VOLUME) ---

// --- User API ---

// User represents a local user in TrueNAS.
type User struct {
	ID               int       `json:"id"`
	UID              int       `json:"uid"`
	Username         string    `json:"username"`
	FullName         string    `json:"full_name"`
	Email            *string   `json:"email"`
	Home             string    `json:"home"`
	Shell            string    `json:"shell"`
	Builtin          bool      `json:"builtin"`
	Locked           bool      `json:"locked"`
	SMB              bool      `json:"smb"`
	SSHPubKey        *string   `json:"sshpubkey"`
	PasswordDisabled bool      `json:"password_disabled"`
	Group            UserGroup `json:"group"`
	Groups           []int     `json:"groups"`
	SudoCommands     []string  `json:"sudo_commands"`
	SudoCommandsNP   []string  `json:"sudo_commands_nopasswd"`
}

// UserGroup represents the primary group of a user.
type UserGroup struct {
	ID    int    `json:"id"`
	GID   int    `json:"bsdgrp_gid"`
	Group string `json:"bsdgrp_group"`
}

// UserCreateRequest represents the request to create a user.
type UserCreateRequest struct {
	Username       string   `json:"username"`
	FullName       string   `json:"full_name"`
	Email          string   `json:"email,omitempty"`
	Password       string   `json:"password"`
	UID            int      `json:"uid,omitempty"`
	Group          int      `json:"group,omitempty"`
	GroupCreate    bool     `json:"group_create"`
	Groups         []int    `json:"groups,omitempty"`
	Home           string   `json:"home,omitempty"`
	Shell          string   `json:"shell,omitempty"`
	Locked         bool     `json:"locked"`
	SMB            bool     `json:"smb"`
	SSHPubKey      string   `json:"sshpubkey,omitempty"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}

// UserUpdateRequest represents the request to update a user.
type UserUpdateRequest struct {
	FullName       string   `json:"full_name,omitempty"`
	Email          string   `json:"email,omitempty"`
	Password       string   `json:"password,omitempty"`
	Group          int      `json:"group,omitempty"`
	Groups         []int    `json:"groups,omitempty"`
	Home           string   `json:"home,omitempty"`
	Shell          string   `json:"shell,omitempty"`
	Locked         *bool    `json:"locked,omitempty"`
	SMB            *bool    `json:"smb,omitempty"`
	SSHPubKey      string   `json:"sshpubkey,omitempty"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}

// GetUser retrieves a user by ID.
func (c *Client) GetUser(ctx context.Context, id int) (*User, error) {
	tflog.Trace(ctx, "GetUser start")

	resp, err := c.Get(ctx, fmt.Sprintf("/user/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting user %d: %w", id, err)
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}

	tflog.Trace(ctx, "GetUser success")
	return &user, nil
}

// CreateUser creates a new local user.
func (c *Client) CreateUser(ctx context.Context, req *UserCreateRequest) (*User, error) {
	tflog.Trace(ctx, "CreateUser start")

	resp, err := c.Post(ctx, "/user", req)
	if err != nil {
		return nil, fmt.Errorf("creating user %q: %w", req.Username, err)
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("parsing user create response: %w", err)
	}

	tflog.Trace(ctx, "CreateUser success")
	return &user, nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, id int, req *UserUpdateRequest) (*User, error) {
	tflog.Trace(ctx, "UpdateUser start")

	resp, err := c.Put(ctx, fmt.Sprintf("/user/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating user %d: %w", id, err)
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("parsing user update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateUser success")
	return &user, nil
}

// DeleteUser deletes a user.
func (c *Client) DeleteUser(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteUser start")

	// TrueNAS user delete requires a body: {"delete_group": true}
	body := map[string]bool{"delete_group": true}
	_, err := c.DeleteWithBody(ctx, fmt.Sprintf("/user/id/%d", id), body)
	if err != nil {
		return fmt.Errorf("deleting user %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteUser success")
	return nil
}

// --- Group API ---

// Group represents a local group in TrueNAS.
type Group struct {
	ID             int      `json:"id"`
	GID            int      `json:"gid"`
	Name           string   `json:"name"`
	Builtin        bool     `json:"builtin"`
	SMB            bool     `json:"smb"`
	SudoCommands   []string `json:"sudo_commands"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd"`
	Users          []int    `json:"users"`
}

// GroupCreateRequest represents the request to create a group.
type GroupCreateRequest struct {
	Name           string   `json:"name"`
	GID            int      `json:"gid,omitempty"`
	SMB            bool     `json:"smb"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}

// GroupUpdateRequest represents the request to update a group.
type GroupUpdateRequest struct {
	Name           string   `json:"name,omitempty"`
	SMB            *bool    `json:"smb,omitempty"`
	SudoCommands   []string `json:"sudo_commands,omitempty"`
	SudoCommandsNP []string `json:"sudo_commands_nopasswd,omitempty"`
}

// GetGroup retrieves a group by ID.
func (c *Client) GetGroup(ctx context.Context, id int) (*Group, error) {
	tflog.Trace(ctx, "GetGroup start")

	resp, err := c.Get(ctx, fmt.Sprintf("/group/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting group %d: %w", id, err)
	}

	var group Group
	if err := json.Unmarshal(resp, &group); err != nil {
		return nil, fmt.Errorf("parsing group response: %w", err)
	}

	tflog.Trace(ctx, "GetGroup success")
	return &group, nil
}

// CreateGroup creates a new local group.
func (c *Client) CreateGroup(ctx context.Context, req *GroupCreateRequest) (*Group, error) {
	tflog.Trace(ctx, "CreateGroup start")

	resp, err := c.Post(ctx, "/group", req)
	if err != nil {
		return nil, fmt.Errorf("creating group %q: %w", req.Name, err)
	}

	// The create endpoint returns just the group ID as an integer.
	var groupID int
	if err := json.Unmarshal(resp, &groupID); err != nil {
		return nil, fmt.Errorf("parsing group create response: %w", err)
	}

	tflog.Trace(ctx, "CreateGroup success")
	return c.GetGroup(ctx, groupID)
}

// UpdateGroup updates an existing group.
func (c *Client) UpdateGroup(ctx context.Context, id int, req *GroupUpdateRequest) (*Group, error) {
	tflog.Trace(ctx, "UpdateGroup start")

	resp, err := c.Put(ctx, fmt.Sprintf("/group/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating group %d: %w", id, err)
	}

	// The update endpoint returns the group ID, not the full object.
	var groupID int
	if err := json.Unmarshal(resp, &groupID); err != nil {
		return nil, fmt.Errorf("parsing group update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateGroup success")
	return c.GetGroup(ctx, groupID)
}

// DeleteGroup deletes a group.
func (c *Client) DeleteGroup(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteGroup start")

	// TrueNAS group delete requires a body: {"delete_users": false}
	body := map[string]bool{"delete_users": false}
	_, err := c.DeleteWithBody(ctx, fmt.Sprintf("/group/id/%d", id), body)
	if err != nil {
		return fmt.Errorf("deleting group %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteGroup success")
	return nil
}

// --- Tunable API ---

// --- Cloud Sync API ---

// --- Rsync Task API ---

// --- Network Configuration API ---

// --- Init/Startup Script API ---

// --- Disk API (read-only) ---

// --- iSCSI Target-Extent Association API ---

// --- Pool Scrub Task API ---

// --- Full Network Configuration Resource API ---

// --- Filesystem ACL API ---
