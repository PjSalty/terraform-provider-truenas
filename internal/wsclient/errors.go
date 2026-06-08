// Package wsclient implements a TrueNAS SCALE client over the JSON-RPC 2.0
// WebSocket transport (the supported API surface from TrueNAS SCALE 25.04
// onwards). The legacy REST client lives in internal/client and is being
// migrated method-by-method; both packages export the same per-resource
// helper signatures so resource code is transport-agnostic.
package wsclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// JSON-RPC 2.0 standard error codes. The TrueNAS server adds two
// implementation-specific codes documented at
// https://api.truenas.com/v25.04/jsonrpc.html — we recognize both.
const (
	// CodeParseError is the JSON-RPC standard "Invalid JSON received"
	// code. We never expect to see this since we control the request
	// shape, but the client maps it through for completeness.
	CodeParseError = -32700
	// CodeInvalidRequest indicates the JSON sent was a valid object but
	// not a valid Request object.
	CodeInvalidRequest = -32600
	// CodeMethodNotFound is returned when the named JSON-RPC method does
	// not exist on the server. This is the canonical "you typed the
	// method name wrong" signal and the wsclient's IsNotFound treats it
	// as a 404-equivalent for resource Read paths.
	CodeMethodNotFound = -32601
	// CodeInvalidParams is returned when params do not match the method
	// signature.
	CodeInvalidParams = -32602
	// CodeInternalError is the catch-all for unexpected server failures.
	CodeInternalError = -32603
	// CodeTooManyConcurrent is the TrueNAS-specific "too many concurrent
	// calls" code. The transport layer treats this as a backpressure
	// signal and waits before retrying.
	CodeTooManyConcurrent = -32000
	// CodeMethodCallError is the TrueNAS-specific code for an exception
	// raised during method execution. The error.data payload carries the
	// underlying middlewared exception (errname, reason, trace).
	CodeMethodCallError = -32001
)

// RPCError is the JSON-RPC 2.0 error envelope returned by TrueNAS when a
// call fails. It implements the error interface so callers can either
// type-assert for the structured fields or just call .Error() for a
// pre-formatted message.
//
// Data carries the TrueNAS-specific failure detail when the server
// returns CodeMethodCallError. The shape is:
//
//	{
//	  "errname": "EINVAL" | "ENOENT" | "EEXIST" | "ValidationErrors" | ...,
//	  "reason":  "Human-readable reason",
//	  "trace":   { ... server-side traceback, only with debug auth },
//	  "py_exception": "..."
//	}
//
// We intentionally keep Data as json.RawMessage rather than a typed
// struct: the trace shape is undocumented and varies between TrueNAS
// firmware revisions, and code that needs a specific field can decode
// it on demand.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error returns a redacted, human-readable form of the RPC failure. The
// surrounding transport layer is responsible for running the message
// through redactMessage before it ever reaches Diagnostics; this method
// just formats what the server returned.
func (e *RPCError) Error() string {
	if e == nil {
		return "<nil RPCError>"
	}
	errname, reason := e.errnameAndReason()
	// Only the `reason` field gets redacted: TrueNAS' middlewared
	// frequently quotes the offending request payload verbatim in
	// reasons (caught by the brutal redactor matrix when a
	// ValidationErrors reason contained
	// "bad value for password: 's3cret'"). `Message` is the static
	// server-controlled label ("Method call error", "cloud sync
	// credential 'foo' not found") and is left intact so legitimate
	// diagnostics still read cleanly — running it through the
	// fragment matcher would strip phrases like "cloud sync
	// credential" because they contain the substring "credential".
	reason = redactMessage(reason)
	switch {
	case errname != "" && reason != "":
		return fmt.Sprintf("truenas rpc error %d: %s: %s (%s)", e.Code, e.Message, reason, errname)
	case errname != "":
		return fmt.Sprintf("truenas rpc error %d: %s (%s)", e.Code, e.Message, errname)
	case reason != "":
		return fmt.Sprintf("truenas rpc error %d: %s: %s", e.Code, e.Message, reason)
	default:
		return fmt.Sprintf("truenas rpc error %d: %s", e.Code, e.Message)
	}
}

// errnameAndReason pulls the `errname` and `reason` fields out of
// e.Data when the server included them. Both default to "" on any
// decode failure — the wrapping Error() call still returns a usable
// message even when the data shape is unexpected.
func (e *RPCError) errnameAndReason() (errname, reason string) {
	if len(e.Data) == 0 {
		return "", ""
	}
	var d struct {
		ErrName string `json:"errname"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return "", ""
	}
	return d.ErrName, d.Reason
}

// IsNotFound reports whether err signals "the resource does not exist
// on the server" — the WebSocket equivalent of HTTP 404. Resources
// call this from their Read path to drop a missing instance from
// state instead of returning an error that would block the run.
//
// Recognized signals:
//
//   - CodeMethodNotFound (-32601): the method itself is unknown. This
//     usually means a typo in the wsclient code, but is also the
//     correct response shape when a resource-specific instance method
//     has been removed in a TrueNAS upgrade. Treat as not-found so
//     terraform state catches up cleanly.
//
//   - CodeMethodCallError (-32001) with errname ∈ {ENOENT, ValidationErrors}
//     and the reason text mentioning "does not exist" / "not found" /
//     "no such" — this is how TrueNAS typically signals a missing
//     instance from a `*.get_instance` or `*.delete` call.
//
// IsNotFound is intentionally conservative: ambiguous server errors
// (network drop, internal error) return false, so the resource's
// CRUD code escalates them rather than silently dropping state.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	if rpcErr.Code == CodeMethodNotFound {
		return true
	}
	if rpcErr.Code != CodeMethodCallError {
		return false
	}
	errname, reason := rpcErr.errnameAndReason()
	if errname == "ENOENT" {
		return true
	}
	low := strings.ToLower(reason)
	if errname == "ValidationErrors" || errname == "EINVAL" {
		// Validation-style "no such object" surfaces here; gate on
		// reason text so we do not swallow legitimate input errors.
		if strings.Contains(low, "does not exist") ||
			strings.Contains(low, "not found") ||
			strings.Contains(low, "no such") {
			return true
		}
	}
	return false
}

// ErrConnectionLost is returned from Call() when the WebSocket
// transport drops mid-request. It is wrapped, not bare, so callers
// can errors.Is(err, ErrConnectionLost) and decide whether to retry.
//
// The transport layer surfaces this even when an auto-reconnect is
// already in flight — callers with idempotent semantics opt back in
// via the CallOptions.Idempotent flag, never by inspecting this
// error directly.
var ErrConnectionLost = errors.New("websocket connection lost")

// ErrShuttingDown is returned from Call() when the Client has been
// closed (typically at provider shutdown). Distinct from
// ErrConnectionLost: the connection is gone deliberately, not
// transiently, so retry is not appropriate.
var ErrShuttingDown = errors.New("websocket client is shutting down")
