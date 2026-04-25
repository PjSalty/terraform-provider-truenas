package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// requiresReplaceInt64 implements planmodifier.Int64 for immutable numeric
// attributes that cannot be changed in-place — any change forces resource
// replacement (destroy + create).
type requiresReplaceInt64 struct {
	description string
}

// RequiresReplaceIfChangedInt64 returns a plan modifier that triggers
// resource replacement whenever the int64 value changes from its prior
// state. Use on numeric identifiers that the upstream API treats as
// immutable (e.g. TrueNAS NVMe-oF subsystem NQN ports that can't be
// renumbered in place).
func RequiresReplaceIfChangedInt64() planmodifier.Int64 {
	return requiresReplaceInt64{
		description: "If the value of this attribute changes, Terraform will destroy and recreate the resource.",
	}
}

func (m requiresReplaceInt64) Description(_ context.Context) string {
	return m.description
}

func (m requiresReplaceInt64) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m requiresReplaceInt64) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
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
