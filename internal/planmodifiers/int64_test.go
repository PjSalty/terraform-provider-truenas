package planmodifiers_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers"
)

func TestRequiresReplaceIfChangedInt64_Changed(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedInt64()
	req := planmodifier.Int64Request{
		StateValue: types.Int64Value(1),
		PlanValue:  types.Int64Value(2),
	}
	resp := &planmodifier.Int64Response{}
	m.PlanModifyInt64(context.Background(), req, resp)
	if !resp.RequiresReplace {
		t.Error("expected RequiresReplace=true when state and plan differ")
	}
}

func TestRequiresReplaceIfChangedInt64_Unchanged(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedInt64()
	req := planmodifier.Int64Request{
		StateValue: types.Int64Value(5),
		PlanValue:  types.Int64Value(5),
	}
	resp := &planmodifier.Int64Response{}
	m.PlanModifyInt64(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("expected RequiresReplace=false when state matches plan")
	}
}

func TestRequiresReplaceIfChangedInt64_NullState(t *testing.T) {
	// Null state means create — no replacement.
	m := planmodifiers.RequiresReplaceIfChangedInt64()
	req := planmodifier.Int64Request{
		StateValue: types.Int64Null(),
		PlanValue:  types.Int64Value(1),
	}
	resp := &planmodifier.Int64Response{}
	m.PlanModifyInt64(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("expected no replacement during create (null state)")
	}
}

func TestRequiresReplaceIfChangedInt64_UnknownPlan(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedInt64()
	req := planmodifier.Int64Request{
		StateValue: types.Int64Value(1),
		PlanValue:  types.Int64Unknown(),
	}
	resp := &planmodifier.Int64Response{}
	m.PlanModifyInt64(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("expected no replacement when plan is unknown")
	}
}

func TestRequiresReplaceIfChangedInt64_Descriptions(t *testing.T) {
	m := planmodifiers.RequiresReplaceIfChangedInt64()
	if d := m.Description(context.Background()); d == "" {
		t.Error("Description must be non-empty")
	}
	if d := m.MarkdownDescription(context.Background()); d == "" {
		t.Error("MarkdownDescription must be non-empty")
	}
}
