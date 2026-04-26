package planmodifiers_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers"
)

func TestRequiresReplaceIfChangedBool_Changed(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedBool()
	req := planmodifier.BoolRequest{
		StateValue: types.BoolValue(false),
		PlanValue:  types.BoolValue(true),
	}
	resp := &planmodifier.BoolResponse{}
	m.PlanModifyBool(context.Background(), req, resp)
	if !resp.RequiresReplace {
		t.Error("expected RequiresReplace=true when state and plan differ")
	}
}

func TestRequiresReplaceIfChangedBool_Unchanged(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedBool()
	req := planmodifier.BoolRequest{
		StateValue: types.BoolValue(true),
		PlanValue:  types.BoolValue(true),
	}
	resp := &planmodifier.BoolResponse{}
	m.PlanModifyBool(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("expected RequiresReplace=false when state matches plan")
	}
}

func TestRequiresReplaceIfChangedBool_NullState(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedBool()
	req := planmodifier.BoolRequest{
		StateValue: types.BoolNull(),
		PlanValue:  types.BoolValue(true),
	}
	resp := &planmodifier.BoolResponse{}
	m.PlanModifyBool(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("expected no replacement during create (null state)")
	}
}

func TestRequiresReplaceIfChangedBool_UnknownPlan(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedBool()
	req := planmodifier.BoolRequest{
		StateValue: types.BoolValue(true),
		PlanValue:  types.BoolUnknown(),
	}
	resp := &planmodifier.BoolResponse{}
	m.PlanModifyBool(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("expected no replacement when plan is unknown")
	}
}

func TestRequiresReplaceIfChangedBool_Descriptions(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedBool()
	if d := m.Description(context.Background()); d == "" {
		t.Error("Description must be non-empty")
	}
	if d := m.MarkdownDescription(context.Background()); d == "" {
		t.Error("MarkdownDescription must be non-empty")
	}
}
