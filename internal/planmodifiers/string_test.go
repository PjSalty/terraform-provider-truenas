package planmodifiers_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	pm "github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers"
)

func TestRequiresReplaceIfChanged_NoState(t *testing.T) {
	// On create (no prior state) replacement must NOT be triggered.
	mod := pm.RequiresReplaceIfChanged()
	req := planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("tank"),
	}
	resp := &planmodifier.StringResponse{}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("RequiresReplace should be false when state is null (resource creation)")
	}
}

func TestRequiresReplaceIfChanged_SameValue(t *testing.T) {
	mod := pm.RequiresReplaceIfChanged()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("tank"),
		PlanValue:  types.StringValue("tank"),
	}
	resp := &planmodifier.StringResponse{}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("RequiresReplace should be false when value is unchanged")
	}
}

func TestRequiresReplaceIfChanged_DifferentValue(t *testing.T) {
	mod := pm.RequiresReplaceIfChanged()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("tank"),
		PlanValue:  types.StringValue("pool2"),
	}
	resp := &planmodifier.StringResponse{}
	mod.PlanModifyString(context.Background(), req, resp)
	if !resp.RequiresReplace {
		t.Error("RequiresReplace should be true when value changes")
	}
}

func TestRequiresReplaceIfChanged_UnknownPlan(t *testing.T) {
	// If the planned value is unknown (computed), do not force replace.
	mod := pm.RequiresReplaceIfChanged()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("tank"),
		PlanValue:  types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Error("RequiresReplace should be false when plan value is unknown")
	}
}

func TestRequiresReplaceIfChanged_Descriptions(t *testing.T) {
	mod := pm.RequiresReplaceIfChanged()
	ctx := context.Background()

	desc := mod.Description(ctx)
	if desc == "" {
		t.Error("Description() returned empty string")
	}

	md := mod.MarkdownDescription(ctx)
	if md == "" {
		t.Error("MarkdownDescription() returned empty string")
	}
	if md != desc {
		t.Errorf("MarkdownDescription should equal Description; got %q vs %q", md, desc)
	}
}
