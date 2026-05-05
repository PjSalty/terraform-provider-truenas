package wsclient

import (
	"errors"
	"fmt"
	"strings"
)

// ErrDestroyProtected is returned from Call() when DestroyProtection is
// true and the JSON-RPC method is classified as destructive. Sibling
// safety rail to ErrReadOnly: where ReadOnly blocks every mutating
// method, DestroyProtection blocks ONLY the destructive ones. Creates
// and updates flow through normally.
//
// Mirrors client.ErrDestroyProtected from the legacy REST package; same
// operator-facing semantics, new transport. The intended use is the
// safe-apply profile — the operator's first real apply against prod
// can create + update but cannot delete until they explicitly opt out.
var ErrDestroyProtected = errors.New("truenas wsclient is in destroy-protected mode: refusing to send destructive call. Set destroy_protection=false (or unset TRUENAS_DESTROY_PROTECTION) to allow destructive operations")

// destructiveMethodSuffixes is the set of method-name suffixes
// classified as destructive. The match is against the last dot-segment
// of the method ("pool.dataset.delete" → "delete"). This catches the
// canonical destructive surface; unusual methods that delete under a
// non-standard name (e.g. "system.reboot", which destroys an in-flight
// session even though it does not call something "delete") need to opt
// in explicitly via CallOptions.Destroys: true.
//
// Conservative on purpose: a missed classification means a destroy
// reaches the server when the operator wanted protection. A false
// positive just means a non-destroy fails closed and the operator has
// to opt out.
var destructiveMethodSuffixes = []string{
	"delete",
	"destroy",
	"remove",
	"unset",
	"purge",
	"wipe",
}

// isDestructiveMethod reports whether method is destructive. Same
// suffix-on-last-segment heuristic as isReadOnlyMethod. Empty method
// strings return false (caller is responsible for not passing them).
func isDestructiveMethod(method string) bool {
	if method == "" {
		return false
	}
	last := method
	if i := strings.LastIndex(method, "."); i >= 0 {
		last = method[i+1:]
	}
	for _, s := range destructiveMethodSuffixes {
		if last == s {
			return true
		}
	}
	return false
}

// checkDestroyProtection returns an ErrDestroyProtected-wrapped error
// when the client is in destroy-protected mode and the method (or
// caller-asserted CallOptions) is destructive, or nil when the
// request should pass. Called from Call() AFTER checkReadOnly; the
// two rails are independent and both enforced.
func (c *Client) checkDestroyProtection(method string, opts CallOptions) error {
	if c == nil || !c.DestroyProtection {
		return nil
	}
	if opts.Destroys {
		return fmt.Errorf("%w: %s", ErrDestroyProtected, method)
	}
	if !isDestructiveMethod(method) {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrDestroyProtected, method)
}
