package planhelpers_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

// fakeSchema is the smallest possible schema the Plugin Framework
// will accept and still be able to construct a State/Plan object.
// We only need the `id` attribute because the warning helper reads
// exactly that one field.
var fakeSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true},
	},
}

func stateWithID(t *testing.T, id string) tfsdk.State {
	t.Helper()
	objType := fakeSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	return tfsdk.State{
		Schema: fakeSchema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id": tftypes.NewValue(tftypes.String, id),
		}),
	}
}

func nullPlan() tfsdk.Plan {
	objType := fakeSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	return tfsdk.Plan{
		Schema: fakeSchema,
		Raw:    tftypes.NewValue(objType, nil),
	}
}

func planWithID(t *testing.T, id string) tfsdk.Plan {
	t.Helper()
	objType := fakeSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	return tfsdk.Plan{
		Schema: fakeSchema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id": tftypes.NewValue(tftypes.String, id),
		}),
	}
}

func nullState() tfsdk.State {
	objType := fakeSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	return tfsdk.State{
		Schema: fakeSchema,
		Raw:    tftypes.NewValue(objType, nil),
	}
}

// TestWarnOnDestroy_DestroyEmitsWarning is the happy path: plan is
// null (destroy), state holds a resource — the helper must emit
// exactly one Warning whose summary names the resource type and ID.
func TestWarnOnDestroy_DestroyEmitsWarning(t *testing.T) {
	req := resource.ModifyPlanRequest{
		Plan:  nullPlan(),
		State: stateWithID(t, "tank/important"),
	}
	resp := &resource.ModifyPlanResponse{Diagnostics: diag.Diagnostics{}}
	planhelpers.WarnOnDestroy(context.Background(), req, resp, "truenas_dataset")

	warns := resp.Diagnostics.Warnings()
	if len(warns) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warns))
	}
	if !strings.Contains(warns[0].Summary(), "truenas_dataset") {
		t.Errorf("warning summary missing resource type: %q", warns[0].Summary())
	}
	if !strings.Contains(warns[0].Summary(), "tank/important") {
		t.Errorf("warning summary missing resource ID: %q", warns[0].Summary())
	}
	if !strings.Contains(warns[0].Detail(), "destroy_protection") {
		t.Errorf("warning detail missing destroy_protection pointer: %q", warns[0].Detail())
	}
	if resp.Diagnostics.HasError() {
		t.Error("WarnOnDestroy must not emit errors, only warnings")
	}
}

// TestWarnOnDestroy_CreateIsNoOp: plan present, state null — this is
// a create. No warning.
func TestWarnOnDestroy_CreateIsNoOp(t *testing.T) {
	req := resource.ModifyPlanRequest{
		Plan:  planWithID(t, "tank/new"),
		State: nullState(),
	}
	resp := &resource.ModifyPlanResponse{Diagnostics: diag.Diagnostics{}}
	planhelpers.WarnOnDestroy(context.Background(), req, resp, "truenas_dataset")
	if len(resp.Diagnostics.Warnings()) != 0 {
		t.Errorf("create action should not emit a warning, got %v", resp.Diagnostics.Warnings())
	}
}

// TestWarnOnDestroy_UpdateIsNoOp: both plan and state present — this
// is an update. No warning.
func TestWarnOnDestroy_UpdateIsNoOp(t *testing.T) {
	req := resource.ModifyPlanRequest{
		Plan:  planWithID(t, "tank/existing"),
		State: stateWithID(t, "tank/existing"),
	}
	resp := &resource.ModifyPlanResponse{Diagnostics: diag.Diagnostics{}}
	planhelpers.WarnOnDestroy(context.Background(), req, resp, "truenas_dataset")
	if len(resp.Diagnostics.Warnings()) != 0 {
		t.Errorf("update action should not emit a warning, got %v", resp.Diagnostics.Warnings())
	}
}

// TestWarnOnDestroy_BothNullNoOp: neither plan nor state — the
// resource is not in the plan at all. No warning.
func TestWarnOnDestroy_BothNullNoOp(t *testing.T) {
	req := resource.ModifyPlanRequest{
		Plan:  nullPlan(),
		State: nullState(),
	}
	resp := &resource.ModifyPlanResponse{Diagnostics: diag.Diagnostics{}}
	planhelpers.WarnOnDestroy(context.Background(), req, resp, "truenas_dataset")
	if len(resp.Diagnostics.Warnings()) != 0 {
		t.Errorf("both-null should not emit a warning, got %v", resp.Diagnostics.Warnings())
	}
}

// TestWarnOnDestroy_EmptyIDFallback exercises the `if id == "" { id =
// "(unknown)" }` branch. State is non-null, plan is null (destroy),
// but the id attribute value is an empty string — the warning should
// still emit with the "(unknown)" placeholder rather than an empty
// quoted string. Closes the last ~11% gap on WarnOnDestroy.
func TestWarnOnDestroy_EmptyIDFallback(t *testing.T) {
	req := resource.ModifyPlanRequest{
		Plan:  nullPlan(),
		State: stateWithID(t, ""),
	}
	resp := &resource.ModifyPlanResponse{Diagnostics: diag.Diagnostics{}}
	planhelpers.WarnOnDestroy(context.Background(), req, resp, "truenas_dataset")

	warns := resp.Diagnostics.Warnings()
	if len(warns) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warns))
	}
	if !strings.Contains(warns[0].Summary(), "(unknown)") {
		t.Errorf("empty-id destroy should render (unknown) in summary: %q", warns[0].Summary())
	}
}
