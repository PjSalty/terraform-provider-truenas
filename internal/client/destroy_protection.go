package client

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrDestroyProtected is returned from any *Client request when
// DestroyProtection is true and the HTTP method is DELETE. It is a
// sibling safety rail to ErrReadOnly: where ReadOnly blocks every
// mutating verb (POST/PUT/DELETE), DestroyProtection blocks ONLY the
// destructive DELETE verb. Creates and updates flow through normally.
//
// This is the AWS-provider-equivalent pattern for `deletion_protection`
// on resources like aws_db_instance, aws_eks_cluster, and aws_lb —
// except it operates at the client layer for EVERY resource in the
// provider at once, so there is no per-resource coverage gap and no
// schema surface to audit.
//
// Intended use: the operator's second apply against a production
// TrueNAS. Phase 0 of phased-rollout.md runs with ReadOnly=true for
// a risk-free dry plan. Phase 1 keeps DestroyProtection=true (default
// for the safe-apply profile) and drops ReadOnly — so the first real
// mutation can create and update resources, but a mis-typed `destroy`
// in the HCL cannot reach the wire. Explicitly set DestroyProtection=
// false (or TRUENAS_DESTROY_PROTECTION=false) only when a destroy is
// intentional and reviewed.
var ErrDestroyProtected = errors.New("truenas client is in destroy-protected mode: refusing to send DELETE request. Set destroy_protection=false (or unset TRUENAS_DESTROY_PROTECTION) to allow destructive operations")

// isDestructiveMethod reports whether the HTTP method is DELETE.
// Kept as a separate helper from isReadOnlyMethod so the two rails
// can evolve independently — a future refinement might add
// HTTP-method pattern matching (e.g. path-based carve-outs for
// api_key.delete that only revokes, never frees resources) without
// touching the read-only gate.
func isDestructiveMethod(method string) bool {
	return method == http.MethodDelete
}

// checkDestroyProtection returns an ErrDestroyProtected-wrapped error
// when c.DestroyProtection is set and method is DELETE, or nil when
// the request should be allowed through. Called from doRequest
// IMMEDIATELY AFTER checkReadOnly — the two rails are independent,
// so a client with both flags set is strictly stricter than either
// alone (read-only blocks POST/PUT/DELETE; destroy protection blocks
// DELETE; the union blocks everything mutating plus specifically
// emphasizes the destructive-path error message).
func (c *Client) checkDestroyProtection(method, path string) error {
	if c == nil || !c.DestroyProtection {
		return nil
	}
	if !isDestructiveMethod(method) {
		return nil
	}
	return fmt.Errorf("%w: %s %s", ErrDestroyProtected, method, path)
}
