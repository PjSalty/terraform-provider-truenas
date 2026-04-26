package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// callModifyPlan drives r.ModifyPlan with a plan/config built from the given
// attribute map. Plan and Config share the same Raw so any cross-attribute
// rule that inspects Config sees the supplied values.
func callModifyPlan(t *testing.T, r resource.ResourceWithModifyPlan, vals map[string]tftypes.Value) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	raw := rawFromValues(t, ctx, sch, vals)

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: sch.Schema, Raw: raw},
		Plan:   tfsdk.Plan{Schema: sch.Schema, Raw: raw},
		State:  tfsdk.State{Schema: sch.Schema, Raw: tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)},
	}
	resp := &resource.ModifyPlanResponse{
		Plan: tfsdk.Plan{Schema: sch.Schema, Raw: raw},
	}
	r.ModifyPlan(ctx, req, resp)
	return resp
}

// callModifyPlanDelete drives r.ModifyPlan with a null plan (delete case).
func callModifyPlanDelete(t *testing.T, r resource.ResourceWithModifyPlan) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	nullRaw := tftypes.NewValue(sch.Schema.Type().TerraformType(ctx), nil)

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: sch.Schema, Raw: nullRaw},
		Plan:   tfsdk.Plan{Schema: sch.Schema, Raw: nullRaw},
		State:  tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)},
	}
	resp := &resource.ModifyPlanResponse{
		Plan: tfsdk.Plan{Schema: sch.Schema, Raw: nullRaw},
	}
	r.ModifyPlan(ctx, req, resp)
	return resp
}
