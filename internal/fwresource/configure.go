// Package fwresource provides cross-cutting helpers for plugin-framework
// resources. Today it hosts the shared Configure plumbing that every
// resource.Resource in internal/resources repeats verbatim: nil-check,
// type-assert *client.Client, emit a diagnostic on mismatch. Extracting
// the pattern here means future resources and review comments can point
// at ConfigureClient instead of copy-pasting the same ten lines.
package fwresource

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// ConfigureClient performs the standard *client.Client type assertion used
// by every resource's Configure method. It returns the client plus an ok
// flag. The caller is expected to stash the client on the resource struct
// when ok is true and otherwise return without further work: on mismatch
// this helper has already appended a descriptive diagnostic to resp.
//
// The nil-ProviderData case (early configure before the provider is
// ready) is treated as a silent non-error: ConfigureClient returns
// (nil, false) without touching resp, matching the framework's own
// expectation that Configure may be called multiple times.
func ConfigureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (*client.Client, bool) {
	if req.ProviderData == nil {
		return nil, false
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return nil, false
	}
	return c, true
}
