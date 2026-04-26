package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// requiresReplaceBool implements planmodifier.Bool for immutable boolean
// attributes that cannot be flipped in-place — any change forces resource
// replacement (destroy + create).
type requiresReplaceBool struct {
	description string
}

// RequiresReplaceIfChangedBool returns a plan modifier that triggers
// resource replacement whenever the bool value changes from its prior
// state. Use on create-time flags like UEFI secure boot toggles that
// upstream APIs reject on update.
func RequiresReplaceIfChangedBool() planmodifier.Bool {
	return requiresReplaceBool{
		description: "If the value of this attribute changes, Terraform will destroy and recreate the resource.",
	}
}

func (m requiresReplaceBool) Description(_ context.Context) string {
	return m.description
}

func (m requiresReplaceBool) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m requiresReplaceBool) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if req.PlanValue.IsUnknown() {
		return
	}
	if !req.StateValue.Equal(req.PlanValue) {
		resp.RequiresReplace = true
	}
}
