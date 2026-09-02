// Package planhelpers holds small helpers shared by resource implementations.
package planhelpers

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Default operation timeouts. These are the values a resource gets when the
// practitioner does not set one, and they are what the documentation says.
//
// Create is the long one: an ACME issuance or a CSR signing can take minutes,
// and a pool or replication operation longer still. Read is short because a
// query that has not answered in a minute is not going to.
const (
	DefaultCreateTimeout = 20 * time.Minute
	DefaultReadTimeout   = 5 * time.Minute
	DefaultUpdateTimeout = 20 * time.Minute
	DefaultDeleteTimeout = 20 * time.Minute
)

// WithCreateTimeout derives a context bounded by the resource's create timeout.
//
// Every resource in this provider declared a timeouts block and none of them
// read it, so a practitioner could write `timeouts { create = "45m" }` and
// change nothing at all. The invariant that guarded it checked only that the
// block was DECLARED, which is presence rather than enforcement.
//
// The returned CancelFunc is always non-nil, so `defer cancel()` is safe even
// when the value could not be read and diagnostics carry the error.
//
// Callers do NOT need to check diagnostics afterwards. An unreadable value
// falls back to the default, so the context is bounded either way, and the
// framework already fails the whole operation on any error-severity
// diagnostic. A per-call-site `if HasError { return }` would add 268 branches
// that no test can reach.
func WithCreateTimeout(ctx context.Context, t timeouts.Value, diags *diag.Diagnostics) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, diags, func() (time.Duration, diag.Diagnostics) {
		return t.Create(ctx, DefaultCreateTimeout)
	}, DefaultCreateTimeout)
}

// WithReadTimeout derives a context bounded by the resource's read timeout.
func WithReadTimeout(ctx context.Context, t timeouts.Value, diags *diag.Diagnostics) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, diags, func() (time.Duration, diag.Diagnostics) {
		return t.Read(ctx, DefaultReadTimeout)
	}, DefaultReadTimeout)
}

// WithUpdateTimeout derives a context bounded by the resource's update timeout.
func WithUpdateTimeout(ctx context.Context, t timeouts.Value, diags *diag.Diagnostics) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, diags, func() (time.Duration, diag.Diagnostics) {
		return t.Update(ctx, DefaultUpdateTimeout)
	}, DefaultUpdateTimeout)
}

// WithDeleteTimeout derives a context bounded by the resource's delete timeout.
func WithDeleteTimeout(ctx context.Context, t timeouts.Value, diags *diag.Diagnostics) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, diags, func() (time.Duration, diag.Diagnostics) {
		return t.Delete(ctx, DefaultDeleteTimeout)
	}, DefaultDeleteTimeout)
}

// withTimeout reads one timeout and bounds the context by it. A value that
// cannot be read falls back to the default rather than leaving the operation
// unbounded: the diagnostic already tells the practitioner what was wrong, and
// an unbounded call is the worse of the two failures.
func withTimeout(
	ctx context.Context,
	diags *diag.Diagnostics,
	read func() (time.Duration, diag.Diagnostics),
	fallback time.Duration,
) (context.Context, context.CancelFunc) {
	d, readDiags := read()
	if diags != nil {
		diags.Append(readDiags...)
	}
	if readDiags.HasError() || d <= 0 {
		d = fallback
	}
	return context.WithTimeout(ctx, d)
}
