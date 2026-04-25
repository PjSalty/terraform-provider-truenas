// Package planmodifiers provides custom terraform-plugin-framework plan modifiers
// for TrueNAS provider resources.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// requiresReplaceString implements planmodifier.String for immutable string
// attributes that cannot be changed in-place — any change forces resource
// replacement (destroy + create).
type requiresReplaceString struct {
	description string
}

// RequiresReplaceIfChanged returns a plan modifier that triggers resource
// replacement whenever the string value changes from its prior state.
// Use this on attributes like pool name and dataset name where TrueNAS
// does not support in-place rename.
func RequiresReplaceIfChanged() planmodifier.String {
	return requiresReplaceString{
		description: "If the value of this attribute changes, Terraform will destroy and recreate the resource.",
	}
}

func (m requiresReplaceString) Description(_ context.Context) string {
	return m.description
}

func (m requiresReplaceString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m requiresReplaceString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Do nothing if there is no prior state (resource is being created).
	if req.StateValue.IsNull() {
		return
	}
	// Do nothing if the planned value is unknown.
	if req.PlanValue.IsUnknown() {
		return
	}
	// Trigger replacement if the value has changed.
	if !req.StateValue.Equal(req.PlanValue) {
		resp.RequiresReplace = true
	}
}
