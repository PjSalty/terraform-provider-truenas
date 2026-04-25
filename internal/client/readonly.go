package client

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrReadOnly is returned from any *Client request when ReadOnly is
// true and the HTTP method is not a safe read (GET/HEAD/OPTIONS).
//
// The gate is enforced BEFORE any network call is made, so the target
// system cannot observe even a dropped mutating request. Intended use:
// point the provider at a production TrueNAS instance with ReadOnly
// enabled, run `terraform plan`, and be physically incapable of
// mutating anything. A failing plan surfaces as a normal Terraform
// error rather than a partially-applied change.
var ErrReadOnly = errors.New("truenas client is in read-only mode: refusing to send mutating request")

// isReadOnlyMethod reports whether the HTTP method is a non-mutating verb
// safe to pass through in read-only mode. PUT and DELETE are idempotent
// but still mutating, so they are blocked just like POST.
func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// checkReadOnly returns an ErrReadOnly-wrapped error when c.ReadOnly is
// set and method is mutating, or nil when the request should be allowed
// through. Keeping this as a method keeps doRequest readable — it can
// just call `if err := c.checkReadOnly(method, path); err != nil`.
func (c *Client) checkReadOnly(method, path string) error {
	if c == nil || !c.ReadOnly {
		return nil
	}
	if isReadOnlyMethod(method) {
		return nil
	}
	return fmt.Errorf("%w: %s %s", ErrReadOnly, method, path)
}
