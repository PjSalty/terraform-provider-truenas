package wsclient

import (
	"errors"
	"fmt"
	"strings"
)

// ErrReadOnly is returned from Call() when ReadOnly is true and the
// JSON-RPC method is classified as mutating. The gate is enforced
// BEFORE the request hits the wire, so the target server cannot
// observe even a dropped mutating request. Mirrors client.ErrReadOnly
// from the legacy REST package, same operator-facing semantics, new
// transport.
var ErrReadOnly = errors.New("truenas wsclient is in read-only mode: refusing to send mutating call")

// readOnlyMethodSuffixes is the set of method-name suffixes that are
// classified as safe reads. The match is on the LAST dot-segment of
// the method (so "pool.dataset.query" and "system.info" both match
// the "query" / "info" suffixes). TrueNAS' JSON-RPC method naming is
// consistent enough that this catches the read surface without per-
// method opt-in.
//
// Anything not in this list is treated as mutating by default. Callers
// that know a method is safe but does not match the pattern (rare -
// mostly utility helpers like "core.ping") can pass CallOptions{Read:
// true} to bypass the gate explicitly.
var readOnlyMethodSuffixes = []string{
	"query",
	"get_instance",
	"get",
	"config",
	"info",
	"check",
	"list",
	"ping",
	"choices",
	"version",
}

// isReadOnlyMethod reports whether method is a safe-read JSON-RPC call.
// The classification is based on the last dot-segment of the method
// name compared against readOnlyMethodSuffixes. An empty method string
// returns false (defensively treated as mutating so a malformed call
// cannot bypass the gate).
func isReadOnlyMethod(method string) bool {
	if method == "" {
		return false
	}
	last := method
	if i := strings.LastIndex(method, "."); i >= 0 {
		last = method[i+1:]
	}
	for _, s := range readOnlyMethodSuffixes {
		if last == s {
			return true
		}
	}
	return false
}

// checkReadOnly returns an ErrReadOnly-wrapped error when the client
// is in read-only mode and the method/options combination is mutating,
// or nil when the request should pass. Called from Call() before any
// network I/O.
//
// The CallOptions.Read explicit override wins over method-name
// classification: a caller can mark a method read-safe even when the
// suffix-based heuristic flags it as mutating. The reverse is also
// true (CallOptions.Mutates: true forces mutating regardless of name)
// to catch the rare server method that mutates state under a "query"-
// style name.
func (c *Client) checkReadOnly(method string, opts CallOptions) error {
	if c == nil || !c.ReadOnly {
		return nil
	}
	if opts.Mutates {
		return fmt.Errorf("%w: %s", ErrReadOnly, method)
	}
	if opts.Read {
		return nil
	}
	if isReadOnlyMethod(method) {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrReadOnly, method)
}
