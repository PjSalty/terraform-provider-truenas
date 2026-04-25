package planmodifiers_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers"
)

func runJSONEquivalent(state, plan string) *planmodifier.StringResponse {
	m := planmodifiers.JSONEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue(state),
		PlanValue:  types.StringValue(plan),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	return resp
}

func TestJSONEquivalent_Identical(t *testing.T) {
	resp := runJSONEquivalent(`{"a":1}`, `{"a":1}`)
	if resp.PlanValue.ValueString() != `{"a":1}` {
		t.Errorf("unchanged byte-identical plan should stay: got %s", resp.PlanValue.ValueString())
	}
}

func TestJSONEquivalent_WhitespaceDiff(t *testing.T) {
	state := `{"a":1}`
	plan := `{  "a":  1  }`
	resp := runJSONEquivalent(state, plan)
	if resp.PlanValue.ValueString() != state {
		t.Errorf("whitespace-only diff should normalize to state %q, got %q", state, resp.PlanValue.ValueString())
	}
}

func TestJSONEquivalent_KeyOrderDiff(t *testing.T) {
	state := `{"a":1,"b":2}`
	// Go's json.Marshal sorts object keys deterministically, so both sides
	// will canonicalize to the same string regardless of input order.
	plan := `{"b":2,"a":1}`
	resp := runJSONEquivalent(state, plan)
	if resp.PlanValue.ValueString() != state {
		t.Errorf("key-order-only diff should normalize to state %q, got %q", state, resp.PlanValue.ValueString())
	}
}

func TestJSONEquivalent_RealDiff(t *testing.T) {
	state := `{"a":1}`
	plan := `{"a":2}`
	resp := runJSONEquivalent(state, plan)
	if resp.PlanValue.ValueString() != plan {
		t.Errorf("real value diff should preserve plan %q, got %q", plan, resp.PlanValue.ValueString())
	}
}

func TestJSONEquivalent_InvalidStateJSON(t *testing.T) {
	// Non-parseable state — modifier should leave plan unchanged.
	state := `not-json`
	plan := `{"a":1}`
	resp := runJSONEquivalent(state, plan)
	if resp.PlanValue.ValueString() != plan {
		t.Errorf("invalid state should not affect plan: got %s", resp.PlanValue.ValueString())
	}
}

func TestJSONEquivalent_InvalidPlanJSON(t *testing.T) {
	state := `{"a":1}`
	plan := `not-json`
	resp := runJSONEquivalent(state, plan)
	if resp.PlanValue.ValueString() != plan {
		t.Errorf("invalid plan should not be normalized: got %s", resp.PlanValue.ValueString())
	}
}

func TestJSONEquivalent_NullState(t *testing.T) {
	m := planmodifiers.JSONEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue(`{"a":1}`),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	if resp.PlanValue.ValueString() != `{"a":1}` {
		t.Errorf("null state should leave plan unchanged: got %s", resp.PlanValue.ValueString())
	}
}

func TestJSONEquivalent_UnknownPlan(t *testing.T) {
	m := planmodifiers.JSONEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue(`{"a":1}`),
		PlanValue:  types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	if !resp.PlanValue.IsUnknown() {
		t.Error("unknown plan should stay unknown")
	}
}

func TestJSONEquivalent_NullPlan(t *testing.T) {
	m := planmodifiers.JSONEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue(`{"a":1}`),
		PlanValue:  types.StringNull(),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	if !resp.PlanValue.IsNull() {
		t.Error("null plan should stay null")
	}
}

func TestJSONEquivalent_Descriptions(t *testing.T) {
	m := planmodifiers.JSONEquivalent()
	if d := m.Description(context.Background()); d == "" {
		t.Error("Description must be non-empty")
	}
	if d := m.MarkdownDescription(context.Background()); d == "" {
		t.Error("MarkdownDescription must be non-empty")
	}
}
